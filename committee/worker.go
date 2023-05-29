package committee

import (
	"errors"
	"go-w3chain/beaconChain"
	"go-w3chain/core"
	"go-w3chain/data"
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

	shardID   uint64
	curHeight *big.Int

	com *Committee
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
		block, err := w.commit(timestamp)
		if err != nil {
			log.Error("worker commit block failed", "err", err)
		}
		timer.Reset(recommit)
		/* 通知committee 有新区块产生 */
		w.InformNewBlock(block)
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
////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////
// worker内部处理交易的函数
//////////////////////////////////////////

/* 生成交易收据, 发送给客户端 */
func (w *worker) sendTXReceipt2Client(txs []*core.Transaction) {
	table := make(map[uint64]*result.TXReceipt)
	for _, tx := range txs {
		if tx.TXStatus == result.DefaultStatus {
			log.Error("record tx status miss!", "tx", tx)
		} else {
			table[tx.ID] = &result.TXReceipt{
				TxID:             tx.ID,
				ConfirmTimeStamp: tx.ConfirmTimestamp,
				TxStatus:         tx.TXStatus,
				ShardID:          int(w.shardID),
				BlockHeight:      w.curHeight.Uint64(),
			}
		}
	}
	w.com.send2Client(table, txs)
	// result.SetTXReceiptV2(table)
}

/**
 * 通知committee 有新区块产生
 * 当committee触发重组时，该方法会被阻塞，进而导致worker被阻塞，直到重组完成
 */
func (w *worker) InformNewBlock(block *core.Block) {
	w.com.NewBlockGenerated(block)
}

/* miner/worker.go:commitWork */
func (w *worker) commit(timestamp int64) (*core.Block, error) {
	stateDB, parentHeight := w.com.getStatusFromShard()
	w.curHeight = parentHeight.Add(parentHeight, common.Big1)
	header := &core.Header{
		Difficulty: math.BigPow(11, 11),
		Number:     w.curHeight,
		Time:       uint64(timestamp),
		ShardID:    uint64(w.shardID),
	}

	txs := w.com.txPool.Pending(w.config.MaxBlockSize)

	w.commitTransactions(txs, stateDB)
	/* commit and insert to blockchain */
	block, err := w.Finalize(header, txs, stateDB)
	if err != nil {
		return nil, errors.New("failed to commit transition state: " + err.Error())
	}

	/* 向信标链记录数据 */
	final_header := block.Header()
	tb := &beaconChain.TimeBeacon{
		Height:     final_header.Number.Uint64(),
		ShardID:    int(w.shardID),
		BlockHash:  block.Hash(),
		TxHash:     final_header.TxHash,
		StatusHash: final_header.Root,
	}

	w.com.AddTB(tb)

	w.com.AddBlock2Shard(block)
	/* 生成交易收据, 并记录到result */
	w.sendTXReceipt2Client(txs)

	log.Debug("create block", "shardID", w.shardID, "block Height", header.Number, "# tx", len(txs), "txpoolLen", w.com.txPool.PendingLen()+w.com.TXpool().PendingRollbackLen())
	// log.Trace("create block", "shardID", w.shardID, "block Height", header.Number, "#TX", len(txs))

	return block, nil

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
	now := time.Now().Unix()
	for _, tx := range txs {
		w.commitTransaction(tx, stateDB, now)
	}
}

func (w *worker) commitTransaction(tx *core.Transaction, stateDB *state.StateDB, now int64) {
	state := stateDB
	tx.TXStatus = result.DefaultStatus
	if tx.TXtype == core.IntraTXType {
		state.SetNonce(*tx.Sender, tx.SenderNonce+1)
		state.SubBalance(*tx.Sender, tx.Value)
		state.AddBalance(*tx.Recipient, tx.Value)
		tx.TXStatus = result.IntraSuccess
		log.Trace("tracing transaction, ", "txid", tx.ID, "status", "committee commit intra tx", "time", now)
	} else if tx.TXtype == core.CrossTXType1 {
		state.SetNonce(*tx.Sender, tx.SenderNonce+1)
		state.SubBalance(*tx.Sender, tx.Value)
		tx.TXStatus = result.CrossTXType1Success
		log.Trace("tracing transaction, ", "txid", tx.ID, "status", "committee commit cross1 tx", "time", now, "tbchain_height", w.com.tbchain_height)
	} else if tx.TXtype == core.CrossTXType2 {
		if w.com.tbchain_height >= tx.ConfirmHeight+tx.RollbackHeight {
			log.Error("This cross2tx is expired, should not be processed by worker",
				"txid", tx.ID,
				"tbchain_cur_height", w.com.tbchain_height,
				"cross1txConfirmHeight", tx.ConfirmHeight,
				"txRollbackHeight", tx.RollbackHeight)
		} else {
			state.AddBalance(*tx.Recipient, tx.Value)
			tx.TXStatus = result.CrossTXType2Success
			log.Trace("tracing transaction, ", "txid", tx.ID, "status", "committee commit cross2 tx", "time", now,
				"tbchain_height", w.com.tbchain_height, "cross1ConfirmHeight", tx.ConfirmHeight, "txRollbackHeight", tx.RollbackHeight)
		}
	} else if tx.TXtype == core.RollbackTXType {
		state.AddBalance(*tx.Sender, tx.Value)
		state.SetNonce(*tx.Sender, tx.SenderNonce-1)
		tx.TXStatus = result.RollbackSuccess
		log.Trace("tracing transaction, ", "txid", tx.ID, "status", "committee commit rollback tx", "time", now)
	} else {
		log.Error("Oops, something wrong! Cannot handle tx type", "cur shardid", w.shardID, "type", tx.TXtype, "tx", tx)
	}
	// tx.ConfirmTimestamp = uint64(now)
}

func (w *worker) Reconfig() {
	log.Info("start reconfiguration...", "before that this committee belongs to shard", w.shardID)

}

/* 委员会1收到回滚交易后，向分片2求证该交易对应的后半部分是否已被打包，若已被打包则不执行回滚交易 */
func (w *worker) checkRollbackTxPacked(tx *core.Transaction) bool {
	shard2ID := data.GetAddrTable()[*tx.Recipient]
	var cross2_packed bool
	callback := func(res ...interface{}) {
		cross2_packed = res[0].(bool)
	}
	w.com.messageHub.Send(core.MsgTypeComGetRollbackProofFromShard, uint64(shard2ID), tx, callback)
	return cross2_packed
}
