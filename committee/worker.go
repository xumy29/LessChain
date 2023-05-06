package committee

import (
	"errors"
	"go-w3chain/beaconChain"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	minRecommitInterval = 5 * time.Second
)

type worker struct {
	config *core.MinerConfig

	// Channels
	startCh chan struct{}
	exitCh  chan struct{}
	// headerCh chan<- struct{} // send to shard

	// atomic status counters
	running int32 // The indicator whether the consensus engine is running or not.

	wg sync.WaitGroup

	messageHub core.MessageHub

	shardID   uint64
	curHeight *big.Int
}

func newWorker(config *core.MinerConfig, shardID uint64) *worker {
	worker := &worker{
		config:  config,
		startCh: make(chan struct{}, 1), // at most 1 element
		exitCh:  make(chan struct{}),
		shardID: shardID,
	}

	// Sanitize recommit interval if the user-specified one is too short.
	recommit := worker.config.Recommit
	if recommit < minRecommitInterval {
		log.Warn("Sanitizing miner recommit interval", "provided", recommit, "updated", minRecommitInterval)
		recommit = minRecommitInterval
	}

	worker.wg.Add(1)
	go worker.newWorkLoop(recommit)

	return worker
}

//////////////////////////////////////////
// worker 生命周期函数
//////////////////////////////////////////

// start sets the running status as 1 and triggers new work submitting.
func (w *worker) start() {
	// log.Info("worker start")
	atomic.StoreInt32(&w.running, 1)
	w.startCh <- struct{}{}
}

// stop sets the running status as 0.
func (w *worker) stop() {
	atomic.StoreInt32(&w.running, 0)
}

// isRunning returns an indicator whether worker is running or not.
func (w *worker) isRunning() bool {
	return atomic.LoadInt32(&w.running) == 1
}

// close terminates all background threads maintained by the worker.
// Note the worker does not support being closed multiple times.
func (w *worker) close() {
	log.Debug("closing worker of this committee..", "shardID", w.shardID)
	w.stop()
	close(w.exitCh)
	w.wg.Wait()
	log.Debug("worker of this committee has been close!", "shardID", w.shardID)
}

// newWorkLoop is a standalone goroutine to submit new sealing work upon received events.
func (w *worker) newWorkLoop(recommit time.Duration) {
	defer w.wg.Done()
	var (
		timestamp int64
	)

	timer := time.NewTimer(0)
	defer timer.Stop()
	<-timer.C // discard the initial tick

	// commit aborts in-flight transaction execution with given signal and resubmits a new one.
	commit := func() {
		if err := w.commit(timestamp); err != nil {
			log.Error("worker commit block failed", "err", err)
		}
		timer.Reset(recommit)
	}

	for {
		select {
		case <-w.exitCh:
			// log.Info("close worker..")
			// log.Debug("worker exitch", "shardID", w.chain.GetChainID())
			return

		case <-w.startCh:
			// log.Debug("worker startch", "shardID", w.chain.GetChainID())
			timestamp = time.Now().Unix()
			commit()

		case <-timer.C:
			// log.Debug("worker timer.c", "shardID", w.chain.GetChainID())
			if w.isRunning() {
				timestamp = time.Now().Unix()
				commit()
			}

			// default:
			// 	log.Debug("worker default", "shardID", w.chain.GetChainID())
			// 	time.Sleep(1000 * time.Millisecond)
		}
	}

}

/*





 */

//////////////////////////////////////////
// worker 通过 messageHub 与外界通信的函数
//////////////////////////////////////////

func (w *worker) SetMessageHub(hub core.MessageHub) {
	w.messageHub = hub
}

/**
 * 向客户端发送交易收据
 * 目前未实现通过网络传输，都是基于messageHub转发
 */
func (w *worker) send2Client(receipts map[uint64]*result.TXReceipt, txs []*core.Transaction) {
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
		w.messageHub.Send(core.MsgTypeShardReply2Client, uint64(cid), msg2Client[cid], nil)
	}
}

/**
* 从对应的分片获取交易和状态
* 按照论文中的设计，此处的状态应该是指交易相关账户的状态以及 merkle proof，
但目前只是把整个状态树的指针传过来，实际上委员会和分片访问和修改的是同一个状态树
*/
func (w *worker) getPoolTxFromShard() ([]*core.Transaction, *state.StateDB, *big.Int) {
	var txs []*core.Transaction
	var states *state.StateDB
	var parentHeight *big.Int
	callback := func(ret ...interface{}) {
		txs = ret[0].([]*core.Transaction)
		states = ret[1].(*state.StateDB)
		parentHeight = ret[2].(*big.Int)
	}
	w.messageHub.Send(core.MsgTypeComGetTX, w.shardID, w.config.MaxBlockSize, callback)
	return txs, states, parentHeight
}

/**
 * 将新区块发送给对应分片
 */
func (w *worker) AddBlock2Shard(block *core.Block) {
	w.messageHub.Send(core.MsgTypeAddBlock2Shard, w.shardID, block, nil)
}

func (w *worker) AddTB(tb *beaconChain.TimeBeacon) {
	w.messageHub.Send(core.MsgTypeAddTB, 0, tb, nil)
}

/*





 */
////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////
// worker内部处理交易的函数
//////////////////////////////////////////

/* 生成交易收据, 发送给客户端并记录到result */
func (w *worker) recordTXReceipt(txs []*core.Transaction) {
	table := make(map[uint64]*result.TXReceipt)
	for _, tx := range txs {
		if tx.TXStatus == result.DefaultStatus {
			log.Warn("record tx status miss!", "tx", tx)
		}
		table[tx.ID] = &result.TXReceipt{
			TxID:             tx.ID,
			ConfirmTimeStamp: tx.ConfirmTimestamp,
			TxStatus:         tx.TXStatus,
			ShardID:          int(w.shardID),
			BlockHeight:      w.curHeight.Uint64(),
		}
	}
	w.send2Client(table, txs)
	result.SetTXReceiptV2(table)
}

/* miner/worker.go:commitWork */
func (w *worker) commit(timestamp int64) error {
	txs, stateDB, parentHeight := w.getPoolTxFromShard()
	w.curHeight = parentHeight.Add(parentHeight, common.Big1)
	header := &core.Header{
		Difficulty: math.BigPow(11, 11),
		Number:     w.curHeight,
		Time:       uint64(timestamp),
		ShardID:    uint64(w.shardID),
	}

	w.commitTransactions(txs, stateDB)
	/* commit and insert to blockchain */
	block, err := w.Finalize(header, txs, stateDB)
	if err != nil {
		return errors.New("failed to commit transition state: " + err.Error())
	}

	/* 向信标链记录数据 */
	final_header := block.Header()
	tb := &beaconChain.TimeBeacon{
		Height:     final_header.Number.Uint64(),
		ShardID:    w.shardID,
		BlockHash:  block.Hash(),
		TxHash:     final_header.TxHash,
		StatusHash: final_header.Root,
	}

	w.AddTB(tb)

	w.AddBlock2Shard(block)
	/* 生成交易收据, 并记录到result */
	w.recordTXReceipt(txs)

	// log.Debug("create block", "block Height", header.Number, "# tx", len(txs), "TxRoot", block.Header().TxHash, "StateRoot:", header.Root)
	// log.Trace("create block", "shardID", w.shardID, "block Height", header.Number, "#TX", len(txs))

	return nil

}

/**
 * 将更新的stateObjects写到MPT树上，得到新树根，并写到区块头中。
 * 根据交易列表得到交易树根，并写到区块头中
 * 根据区块头和交易列表构造区块
 */
func (w *worker) Finalize(header *core.Header, txs []*core.Transaction, stateDB *state.StateDB) (*core.Block, error) {
	state := stateDB
	hashroot, err := state.Commit(false)
	if err != nil {
		return nil, err
	}
	header.Root = hashroot
	block := core.NewBlock(header, txs, trie.NewStackTrie(nil))
	return block, nil

}

/*
* 执行打包的交易，更新stateObjects
 */
func (w *worker) commitTransactions(txs []*core.Transaction, stateDB *state.StateDB) {
	for _, tx := range txs {
		w.commitTransaction(tx, stateDB)
	}
}

func (w *worker) commitTransaction(tx *core.Transaction, stateDB *state.StateDB) {
	state := stateDB
	now := time.Now().Unix()
	tx.TXStatus = result.DefaultStatus
	if tx.TXtype == core.IntraTXType {
		state.SetNonce(*tx.Sender, tx.SenderNonce+1)
		state.SubBalance(*tx.Sender, tx.Value)
		state.AddBalance(*tx.Recipient, tx.Value)
		tx.TXStatus = result.IntraSuccess
	} else if tx.TXtype == core.CrossTXType1 {
		state.SetNonce(*tx.Sender, tx.SenderNonce+1)
		state.SubBalance(*tx.Sender, tx.Value)
		tx.TXStatus = result.CrossTXType1Success
	} else if tx.TXtype == core.CrossTXType2 {
		// tx.ConfirmTimestamp == confirm time of cross1
		// +5 是防止取交易时未超时，执行交易时却超时。增加了一点弹性
		if now > int64(tx.ConfirmTimestamp)+int64(tx.RollbackSecs)+5 {
			log.Warn("This cross2tx is expired, should not be processed by worker", "txid", tx.ID)
		} else {
			state.AddBalance(*tx.Recipient, tx.Value)
			tx.TXStatus = result.CrossTXType2Success
		}
	} else if tx.TXtype == core.RollbackTXType {
		state.AddBalance(*tx.Sender, tx.Value)
		state.SetNonce(*tx.Sender, tx.SenderNonce-1)
		tx.TXStatus = result.RollbackSuccess
	} else {
		log.Error("Oops, something wrong! Cannot handle tx type", "cur shardid", w.shardID, "type", tx.TXtype, "tx", tx)
	}
	tx.ConfirmTimestamp = uint64(now)
}
