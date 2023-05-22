package committee

import (
	"go-w3chain/beaconChain"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"math/big"

	"github.com/ethereum/go-ethereum/core/state"
)

type Committee struct {
	shardID    uint64
	config     *core.MinerConfig
	worker     *worker
	messageHub core.MessageHub
	/* 接收重组结果的管道 */
	reconfigCh chan int
}

func NewCommittee(shardID uint64, config *core.MinerConfig) *Committee {
	worker := newWorker(config, shardID)
	com := &Committee{
		shardID:    shardID,
		config:     config,
		worker:     worker,
		reconfigCh: make(chan int, 1),
	}
	log.Info("NewCommittee", "shardID", shardID)
	worker.com = com

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
		previousID := com.shardID
		com.messageHub.Send(core.MsgTypeReady4Reconfig, com.shardID, nil, nil)
		// 阻塞，直到重组完成，messageHub向管道内发送此委员会所属的新分片ID
		reconfigRes := <-com.reconfigCh
		com.shardID = uint64(reconfigRes)
		com.worker.shardID = uint64(reconfigRes)
		log.Debug("committee reconfiguration done!", "previous shardID", previousID, "now ShardID", com.shardID)
	}
}

func (com *Committee) SetReconfigRes(res int) {
	com.reconfigCh <- res
}

/*





 */

//////////////////////////////////////////
// committee 通过 messageHub 与外界通信的函数
//////////////////////////////////////////

func (com *Committee) SetMessageHub(hub core.MessageHub) {
	com.messageHub = hub
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
		msg2Client[cid] = append(msg2Client[cid], receipts[tx.ID])
	}
	for cid := range msg2Client {
		com.messageHub.Send(core.MsgTypeShardReply2Client, uint64(cid), msg2Client[cid], nil)
	}
}

/**
* 从对应的分片获取交易和状态
* 按照论文中的设计，此处的状态应该是指交易相关账户的状态以及 merkle proof，
但目前只是把整个状态树的指针传过来，实际上委员会和分片访问和修改的是同一个状态树
*/
func (com *Committee) getPoolTxFromShard() ([]*core.Transaction, *state.StateDB, *big.Int) {
	var txs []*core.Transaction
	var states *state.StateDB
	var parentHeight *big.Int
	callback := func(ret ...interface{}) {
		txs = ret[0].([]*core.Transaction)
		states = ret[1].(*state.StateDB)
		parentHeight = ret[2].(*big.Int)
	}
	com.messageHub.Send(core.MsgTypeComGetTX, com.shardID, com.config.MaxBlockSize, callback)
	return txs, states, parentHeight
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
func (com *Committee) AddTB(tb *beaconChain.TimeBeacon) {
	com.messageHub.Send(core.MsgTypeAddTB, 0, tb, nil)
}

/*


 */

//////////////////////////////////////////////////
func (com *Committee) GetShardID() uint64 {
	return com.shardID
}
