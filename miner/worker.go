package miner

import (
	"errors"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"go-w3chain/utils"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	minRecommitInterval = 5 * time.Second
)

type worker struct {
	config *Config
	chain  *core.BlockChain
	pool   *core.TxPool

	// Channels
	startCh  chan struct{}
	exitCh   chan struct{}
	headerCh chan<- struct{} // send to shard

	// atomic status counters
	running int32 // The indicator whether the consensus engine is running or not.

	wg sync.WaitGroup

	MessageHub core.MessageHub
}

func newWorker(config *Config, chain *core.BlockChain, pool *core.TxPool, headerCh chan<- struct{}) *worker {
	worker := &worker{
		config: config,
		chain:  chain,
		pool:   pool,

		startCh:  make(chan struct{}, 1), // at most 1 element
		exitCh:   make(chan struct{}),
		headerCh: headerCh,
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
	log.Debug("closing worker (for mining) of this shard..", "shardID", w.chain.GetChainID())
	atomic.StoreInt32(&w.running, 0)
	close(w.exitCh)
	w.wg.Wait()
	log.Debug("worker (for mining) of this shard has been close!", "shardID", w.chain.GetChainID())
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

func (w *worker) GetAddrTable() map[common.Address]int {
	aInfo := w.GetWorkerAddrInfo()
	return aInfo.AddrTable
}

func (w *worker) GetWorkerAddrInfo() *utils.AddressInfo {
	return w.chain.GetBlockChainAddrInfo()
}

func (w *worker) fillTransactions() []*core.Transaction {
	txs := make([]*core.Transaction, 0)

	/* get normal tx from pending, with resting number txs can be added */
	parHeight := w.chain.GetChainHeight()
	curHeight := parHeight + 1
	txs2 := w.pool.Pending(w.config.MaxBlockSize-len(txs), curHeight)
	txs = append(txs, txs2...)
	return txs
}

func (w *worker) recordTXReceipt(txs []*core.Transaction) {
	table := make(map[uint64]*result.TXReceipt)
	shardID := w.chain.GetChainID()
	for _, tx := range txs {
		if tx.TXStatus == result.DefaultStatus {
			log.Warn("record tx status miss!", "tx", tx)
		}
		table[tx.ID] = &result.TXReceipt{
			TxID:             tx.ID,
			ConfirmTimeStamp: tx.ConfirmTimestamp,
			TxStatus:         tx.TXStatus,
			ShardID:          shardID,
		}
	}
	w.send2Client(table, txs)
	result.SetTXReceiptV2(table)
}

/* miner/worker.go:commitWork */
func (w *worker) commit(timestamp int64) error {
	parent := w.chain.CurrentBlock()
	num := parent.Number()

	header := &core.Header{
		Difficulty: math.BigPow(11, 11),
		Number:     num.Add(num, common.Big1),
		Time:       uint64(timestamp),
		ShardID:    uint64(w.chain.GetChainID()),
	}

	/* fillTransactions from TXpool */
	txs := w.fillTransactions()

	/* 空块跳过 */
	// if len(txs) == 0 {
	// 	log.Debug("skip empty block", "shardID", w.chain.GetChainID(), "block Height", header.Number)
	// 	return nil
	// }

	w.commitTransactions(txs)
	/* commit and insert to blockchain */
	block, err := w.Finalize(header, txs)
	if err != nil {
		return errors.New("failed to commit transition state: " + err.Error())
	}

	/* 生成交易收据, 并记录到result */
	w.recordTXReceipt(txs)
	w.chain.WriteBlock(block)
	// send to shard
	w.headerCh <- struct{}{}
	// log.Debug("create block", "block Height", header.Number, "# tx", len(txs), "TxRoot", block.Header().TxHash, "StateRoot:", header.Root)
	log.Trace("create block", "shardID", w.chain.GetChainID(), "block Height", header.Number, "#TX", len(txs))

	return nil

}

/*
* 将更新的stateObjects写到MPT树上，得到新树根，并写到区块头中。
* 根据区块头和交易列表构造区块
 */
func (w *worker) Finalize(header *core.Header, txs []*core.Transaction) (*core.Block, error) {
	state := w.chain.GetStateDB()
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
func (w *worker) commitTransactions(txs []*core.Transaction) {
	for _, tx := range txs {
		w.commitTransaction(tx)
	}
}

func (w *worker) commitTransaction(tx *core.Transaction) {
	state := w.chain.GetStateDB()
	cur_sid := w.chain.GetChainID()
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
		if now > int64(tx.ConfirmTimestamp)+int64(tx.RollbackSecs) {
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
		log.Error("Oops, something wrong! Cannot handle tx type", "cur shardid", cur_sid, "type", tx.TXtype, "tx", tx)
	}
	tx.ConfirmTimestamp = uint64(now)
}

/* 向客户端发送回复消息
目前未实现通过网络发送
*/
func (w *worker) send2Client(receipts map[uint64]*result.TXReceipt, txs []*core.Transaction) {
	// // 分类型
	// intraReceipts := make([]*result.TXReceipt, 0, len(receipts))
	// cross1Receipts := make([]*result.TXReceipt, 0, len(receipts))
	// cross2Receipts := make([]*result.TXReceipt, 0, len(receipts))
	// for _, tx := range txs {
	// 	switch tx.TXStatus {
	// 	case result.IntraSuccessful:
	// 		intraReceipts = append(intraReceipts, receipts[tx.ID])
	// 	case result.CrossTXType1Successful:
	// 		cross1Receipts = append(cross1Receipts, receipts[tx.ID])
	// 	case result.CrossTXType2Successful:
	// 		cross2Receipts = append(cross2Receipts, receipts[tx.ID])
	// 	}
	// }

	// 分客户端
	msg2Client := make(map[int][]*result.TXReceipt)
	for _, tx := range txs {
		cid := int(tx.Cid)
		if _, ok := msg2Client[cid]; !ok {
			msg2Client[cid] = make([]*result.TXReceipt, 0, len(receipts))
		}
		msg2Client[cid] = append(msg2Client[cid], receipts[tx.ID])
	}
	for cid, _ := range msg2Client {
		w.MessageHub.Send(core.MsgTypeShardReply2Client, cid, msg2Client[cid])
	}
}
