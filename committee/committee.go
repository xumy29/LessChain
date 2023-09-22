package committee

import (
	"bytes"
	"fmt"
	"go-w3chain/beaconChain"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/node"
	"go-w3chain/result"
	"go-w3chain/utils"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
)

type MultiSignData struct {
	MultiSignDone chan struct{}
	Vrfs          [][]byte
	Sigs          [][]byte
	Signers       []common.Address
}

type Committee struct {
	comID      uint32
	config     *core.CommitteeConfig
	worker     *Worker
	messageHub core.MessageHub

	multiSignData *MultiSignData

	/* 接收重组结果的管道 */
	reconfigCh chan []*node.Node
	Nodes      []*node.Node // 同属一个委员会的节点
	members    []*core.NodeInfo
	Node       *node.Node // 当前节点
	txPool     *TxPool
	/* 计数器，初始等于客户端个数，每一个客户端发送注入完成信号时计数器减一 */
	injectNotDone  int32
	tbchain_height uint64
	to_reconfig    bool // 收到特定高度的信标链区块后设为true，准备重组

	shardSendStateChan chan *core.ShardSendState
}

func NewCommittee(comID uint32, clientCnt int, _node *node.Node, config *core.CommitteeConfig) *Committee {
	com := &Committee{
		comID:              comID,
		config:             config,
		multiSignData:      &MultiSignData{},
		reconfigCh:         make(chan []*node.Node, 0),
		Nodes:              []*node.Node{_node},
		members:            []*core.NodeInfo{_node.GetPbftNode().NodeInfo},
		Node:               _node,
		injectNotDone:      int32(clientCnt),
		to_reconfig:        false,
		shardSendStateChan: make(chan *core.ShardSendState, 0),
	}
	log.Info("NewCommittee", "comID", comID, "nodeID", _node.NodeID)

	return com
}

func (com *Committee) Start(nodeId int) {
	if utils.IsComLeader(nodeId) { // 只有委员会的leader节点会运行worker，即出块
		pool := NewTxPool(com.comID)
		com.txPool = pool
		pool.setCommittee(com)

		worker := newWorker(com.config, com.comID)
		com.worker = worker
		worker.setCommittee(com)
	}
}

func (com *Committee) StartWorker() {
	com.worker.start()
}

func (com *Committee) Close() {
	// 避免退出时因reconfigCh阻塞导致worker无法退出
	if com.to_reconfig {
		com.reconfigCh <- nil
	}

	if com.worker != nil {
		com.worker.close()
	}
}

func (com *Committee) SetInjectTXDone(cid uint32) {
	atomic.AddInt32(&com.injectNotDone, -1)
}

/* 交易注入完成且交易池空即可停止 */
func (com *Committee) CanStopV1() bool {
	return com.CanStopV2() && com.txPool.Empty()
}

/* 交易注入完成即可停止 */
func (com *Committee) CanStopV2() bool {
	return com.injectNotDone == 0
}

/**
 * 刚出完一个块判断是否达到重组条件
 * 当committee触发重组时，会在该方法会被阻塞，直到重组完成
 */
func (com *Committee) NewBlockGenerated(block *core.Block) {
	if com.to_reconfig {
		previousNodes := utils.GetFieldValueforList(com.Nodes, "NodeID")

		com.SendReconfigMsg()

		// 阻塞，直到重组完成，messageHub向管道内发送此委员会的新节点
		reconfigRes := <-com.reconfigCh
		com.Reconfig(reconfigRes)

		log.Debug("committee reconfiguration done!",
			"comID", com.comID,
			"previous nodeIDs", previousNodes,
			"now nodeIDs", utils.GetFieldValueforList(com.Nodes, "NodeID"),
		)
		com.to_reconfig = false
	}
}

func (com *Committee) Reconfig(nodes []*node.Node) {
	com.Nodes = nodes

	// 发起交易调用合约，更新委员会节点地址列表
	addrs := make([]common.Address, len(nodes))
	vrfs := make([][]byte, len(nodes))
	for i, node := range nodes {
		addrs[i] = *node.GetAccount().GetAccountAddress()
		vrfs[i] = node.VrfValue
	}
	com.AdjustRecordedAddrs(addrs, vrfs, com.tbchain_height)

	// // 委员会重组后，清空交易池，并标记被丢弃的交易
	// com.txPool = com.txPool.Reset()
}

func (com *Committee) TXpool() *TxPool {
	return com.txPool
}

/*





 */

//////////////////////////////////////////
// committee 通过 messageHub 与外界通信的函数，以及被 messageHub 调用的函数
//////////////////////////////////////////

func (com *Committee) SetMessageHub(hub core.MessageHub) {
	com.messageHub = hub
}

/** 信标链主动向委员会推送新确认的信标时调用此函数
 * 一般情况下信标链应该只向委员会推送其关注的分片和高度的信标，这里进行了简化，默认全部推送
 */
func (com *Committee) AddTBs(tbblock *beaconChain.TBBlock) {
	// for shardID, tbs := range tbs_new {
	// 	for _, tb := range tbs {
	// 		c.tbs[shardID][tb.Height] = tb
	// 	}
	// }
	log.Debug(fmt.Sprintf("committee get tbchain confirm block... %v", tbblock))
	com.tbchain_height = tbblock.Height

	// 收到特定高度的信标链区块后准备重组
	if com.tbchain_height > 0 && com.tbchain_height%uint64(com.config.Height2Reconfig) == 0 {
		com.to_reconfig = true
	}
}

/* 向信标链发起交易，更新委员会地址列表
 */
func (com *Committee) AdjustRecordedAddrs(addrs []common.Address, vrfs [][]byte, seedHeight uint64) {
	data := &AdjustAddrs{
		Addrs:      addrs,
		Vrfs:       vrfs,
		SeedHeight: seedHeight,
	}
	com.messageHub.Send(core.MsgTypeCommitteeAdjustAddrs, com.comID, data, nil)
}

type AdjustAddrs struct {
	Addrs      []common.Address
	Vrfs       [][]byte
	SeedHeight uint64
}

func (com *Committee) HandleClientSendtx(txs []*core.Transaction) {
	com.txPool.AddTxs(txs)
}

/**
 * 向客户端发送交易收据
 * 目前未实现通过网络传输，都是基于messageHub转发
 */
func (com *Committee) send2Client(receipts map[uint64]*result.TXReceipt, txs []*core.Transaction) {
	// 分客户端
	msg2Client := make(map[int][]*result.TXReceipt)
	for _, tx := range txs {
		cid := int(tx.Cid)
		if _, ok := msg2Client[cid]; !ok {
			msg2Client[cid] = make([]*result.TXReceipt, 0, len(receipts))
		}
		if r, ok := receipts[tx.ID]; ok {
			msg2Client[cid] = append(msg2Client[cid], r)
		}
	}
	for cid := range msg2Client {
		com.messageHub.Send(core.MsgTypeCommitteeReply2Client, uint32(cid), msg2Client[cid], nil)
	}
}

func (com *Committee) getBlockHeight() *big.Int {
	var blockHeight *big.Int
	callback := func(res ...interface{}) {
		blockHeight = res[0].(*big.Int)
	}

	data := &core.ComGetHeight{
		From_comID:     com.comID,
		Target_shardID: com.comID,
	}
	com.messageHub.Send(core.MsgTypeComGetHeightFromShard, com.comID, data, callback)

	return blockHeight
}

/* 从对应的分片获取账户状态和证明
 */
func (com *Committee) getStatusFromShard(addrList []common.Address) *core.ShardSendState {
	request := &core.ComGetState{
		From_comID:     com.comID,
		Target_shardID: com.comID,
		AddrList:       addrList, // TODO: implement it
	}
	com.messageHub.Send(core.MsgTypeComGetStateFromShard, com.comID, request, nil)

	response := <-com.shardSendStateChan

	// Validate Merkle proofs for each address
	rootHash := response.StatusTrieHash

	for address, accountData := range response.AccountData {
		proofDB := &proofReader{proof: response.AccountsProofs[address]}

		computedValue, err := trie.VerifyProof(rootHash, utils.GetHash(address.Bytes()), proofDB)
		if err != nil {
			log.Error("Failed to verify Merkle proof", "err", err, "address", address)
			return nil
		}
		if !bytes.Equal(computedValue, accountData) {
			log.Error("Merkle proof verification failed for address", "address", address)
			return nil
		}
	}

	log.Info("getStatusFromShard and verify merkle proof succeed.")

	return response
}

func (com *Committee) HandleShardSendState(response *core.ShardSendState) {
	com.shardSendStateChan <- response
}

/**
 * 将新区块发送给对应分片
 */
func (com *Committee) AddBlock2Shard(block *core.Block) {
	comSendBlock := &core.ComSendBlock{
		Block: block,
	}
	com.messageHub.Send(core.MsgTypeSendBlock2Shard, com.comID, comSendBlock, nil)
}

/**
 * 将新区块的信标发送到信标链
 */
func (com *Committee) SendTB(tb *core.SignedTB) {
	com.messageHub.Send(core.MsgTypeComAddTb2TBChain, 0, tb, nil)
}

func (com *Committee) SendReconfigMsg() {
	com.messageHub.Send(core.MsgTypeReady4Reconfig, com.comID, com.tbchain_height, nil)
}

func (com *Committee) SetReconfigRes(res []*node.Node) {
	com.reconfigCh <- res
}

func (com *Committee) GetEthChainBlockHash(height uint64) (common.Hash, uint64) {
	channel := make(chan struct{}, 1)
	var blockHash common.Hash
	var got_height uint64
	callback := func(ret ...interface{}) {
		blockHash = ret[0].(common.Hash)
		got_height = ret[1].(uint64)
		channel <- struct{}{}
	}
	com.messageHub.Send(core.MsgTypeGetBlockHashFromEthChain, com.comID, height, callback)
	// 阻塞
	<-channel

	return blockHash, got_height
}

/*
	////////////////////////////////////////////////

	///////////////////////////////////////////////

*/

func (com *Committee) GetCommitteeID() uint32 {
	return com.comID
}

func (com *Committee) GetMembers() []*core.NodeInfo {
	return com.members
}

func (com *Committee) AddMember(nodeInfo *core.NodeInfo) {
	com.members = append(com.members, nodeInfo)
}
