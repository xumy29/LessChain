package committee

import (
	"go-w3chain/beaconChain"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"go-w3chain/utils"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/core/state"
)

type Committee struct {
	shardID    uint64
	config     *core.CommitteeConfig
	worker     *worker
	messageHub core.MessageHub
	/* 接收重组结果的管道 */
	reconfigCh chan []*core.Node
	Nodes      []*core.Node
	txPool     *TxPool
	/* 计数器，初始等于客户端个数，每一个客户端发送注入完成信号时计数器减一 */
	injectNotDone  int32
	tbchain_height uint64
}

func NewCommittee(shardID uint64, clientCnt int, nodes []*core.Node, config *core.CommitteeConfig) *Committee {
	worker := newWorker(config, shardID)
	pool := NewTxPool(int(shardID))
	com := &Committee{
		shardID:       shardID,
		config:        config,
		worker:        worker,
		reconfigCh:    make(chan []*core.Node, 1),
		Nodes:         nodes,
		txPool:        pool,
		injectNotDone: int32(clientCnt),
	}
	log.Info("NewCommittee", "shardID", shardID, "nodeIDs", utils.GetFieldValueforList(nodes, "NodeID"))
	worker.com = com
	pool.com = com

	return com
}

func (com *Committee) Start() {
	com.worker.start()
}

func (com *Committee) Close() {
	com.worker.close()
}

/**
 * 判断是否达到重组条件
 * 当committee触发重组时，该方法会被阻塞，直到重组完成
 */
func (com *Committee) NewBlockGenerated(block *core.Block) {
	height := block.NumberU64()
	if height > 0 && height%uint64(com.config.Height2Reconfig) == 0 {
		previousNodes := utils.GetFieldValueforList(com.Nodes, "NodeID")

		com.SendReconfigMsg()

		// 阻塞，直到重组完成，messageHub向管道内发送此委员会的新节点
		reconfigRes := <-com.reconfigCh
		com.Reconfig(reconfigRes)

		log.Debug("committee reconfiguration done!",
			"shardID", com.shardID,
			"previous nodeIDs", previousNodes,
			"now nodeIDs", utils.GetFieldValueforList(com.Nodes, "NodeID"),
		)
	}
}

func (com *Committee) Reconfig(nodes []*core.Node) {
	com.Nodes = nodes
	// 委员会重组后，清空交易池，并标记被丢弃的交易
	com.txPool = com.txPool.Reset()
}

func (com *Committee) InjectTXs(txs []*core.Transaction) {
	com.txPool.AddTxs(txs)
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
 * 委员会收到新确认信标后，
 */
func (com *Committee) AddTBs(tbs_new map[uint32][]*beaconChain.ConfirmedTB, height uint64) {
	// for shardID, tbs := range tbs_new {
	// 	for _, tb := range tbs {
	// 		c.tbs[shardID][tb.Height] = tb
	// 	}
	// }
	com.tbchain_height = height

}

func (com *Committee) SetInjectTXDone() {
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
		com.messageHub.Send(core.MsgTypeCommitteeReply2Client, uint64(cid), msg2Client[cid], nil)
	}
}

/**
* 从对应的分片获取交易和状态
* 按照论文中的设计，此处的状态应该是指交易相关账户的状态以及 merkle proof，
但目前只是把整个状态树的指针传过来，实际上委员会和分片访问和修改的是同一个状态树
*/
func (com *Committee) getStatusFromShard() (*state.StateDB, *big.Int) {
	var states *state.StateDB
	var parentHeight *big.Int
	callback := func(ret ...interface{}) {
		states = ret[0].(*state.StateDB)
		parentHeight = ret[1].(*big.Int)
	}
	com.messageHub.Send(core.MsgTypeComGetStateFromShard, com.shardID, nil, callback)
	return states, parentHeight
}

/**
 * 将新区块发送给对应分片
 */
func (com *Committee) AddBlock2Shard(block *core.Block) {
	com.messageHub.Send(core.MsgTypeAddBlock2Shard, com.shardID, block, nil)
}

/**
 * 将新区块的信标发送到信标链
 */
func (com *Committee) AddTB(tb *beaconChain.SignedTB) {
	com.messageHub.Send(core.MsgTypeCommitteeAddTB, 0, tb, nil)
}

func (com *Committee) SendReconfigMsg() {
	com.messageHub.Send(core.MsgTypeReady4Reconfig, com.shardID, nil, nil)
}

func (com *Committee) SetReconfigRes(res []*core.Node) {
	com.reconfigCh <- res
}

/*
	////////////////////////////////////////////////

	///////////////////////////////////////////////

*/

func (com *Committee) GetShardID() uint64 {
	return com.shardID
}
