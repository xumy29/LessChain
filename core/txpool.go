package core

/*
本地交易: 在本地磁盘存储已发送的交易。这样，本地交易不会丢失，重启节点时可以重新加载到交易池，实时广播出去。
*/

import (
	"go-w3chain/log"
	"go-w3chain/result"
	"sync"
	"time"
)

type TxPool struct {
	// chain blockChain
	shardID int

	pending []*Transaction
	lock    sync.Mutex

	pendingRollback []*Transaction
	r_lock          sync.Mutex
}

func NewTxPool(shardID int) *TxPool {
	pool := &TxPool{
		shardID: shardID,
	}
	return pool
}

/* 重置交易池，返回新的交易池 */
func (pool *TxPool) Reset() *TxPool {
	// 标记交易池中剩下的交易为 dropped
	txs := append(pool.pending, pool.pendingRollback...)
	table := make(map[uint64]*result.TXReceipt)
	for _, tx := range txs {
		tx.TXStatus = result.Dropped
		table[tx.ID] = &result.TXReceipt{
			TxID:     tx.ID,
			TxStatus: tx.TXStatus,
			ShardID:  int(pool.shardID),
		}
	}
	result.SetTXReceiptV2(table)

	// 生成新的交易池
	return NewTxPool(pool.shardID)

}

/* 向交易池中添加交易，调用此方法的方法必须加锁 */
func (pool *TxPool) AddTxWithoutLock(origintx *Transaction) {
	// TODO: 检查本地是否已存有该交易
	if origintx.TXtype == RollbackTXType {
		pool.pendingRollback = append(pool.pendingRollback, origintx)
	} else {
		pool.pending = append(pool.pending, origintx)
	}
}

func (pool *TxPool) AddTxs(txs []*Transaction) {
	/* 需要对lock和r_lock都加锁的场景，都按照先lock再r_lock的顺序，避免死锁 */
	pool.lock.Lock()
	defer pool.lock.Unlock()
	pool.r_lock.Lock()
	defer pool.r_lock.Unlock()
	// fmt.Printf("收到交易%v\n", txs)
	for _, tx := range txs {
		pool.AddTxWithoutLock(tx)
	}

	log.Debug("TxPoolAddTXs", "shardID", pool.shardID, "txPoolPendingLen", pool.PendingLen())
}

/* worker.commitTransaction 从队列取出交易 */
func (pool *TxPool) Pending(maxBlockSize int) []*Transaction {
	/* 需要对lock和r_lock都加锁的场景，都按照先lock再r_lock的顺序，避免死锁 */
	pool.lock.Lock()
	defer pool.lock.Unlock()
	pool.r_lock.Lock()
	defer pool.r_lock.Unlock()
	/* 取交易 */
	txs := make([]*Transaction, 0)
	i := 0

	// 从pendingRollback队列取交易，rollback交易优先于其他交易
	for {
		if i == maxBlockSize || i >= len(pool.pendingRollback) {
			break
		}
		tx := pool.pendingRollback[i]
		txs = append(txs, tx)
		i++
	}
	pool.pendingRollback = pool.pendingRollback[i:]

	// 从pending队列取交易
	maxBlockSize -= i
	i = 0
	pendinglen := len(pool.pending)
	now := uint64(time.Now().Unix())
	for {
		if i == maxBlockSize || i >= pendinglen {
			break
		}
		tx := pool.pending[i]
		// 为保证交易原子性，cross2 交易应判断是否超时
		if tx.TXtype == CrossTXType2 {
			// tx.ConfirmTimestamp是cross1交易confirm的时间
			// now+1 是为了避免从交易池里选交易时时间未超过，commit即填充cross2的confirmTimeStamp时却超过了
			if now > tx.ConfirmTimestamp+tx.RollbackSecs {
				// 如果cross2交易已超时，不会选择该交易进行打包，队列指针后移时需要将maxBlockSize也+1
				i++
				maxBlockSize++
				continue
			}
		}
		txs = append(txs, tx)
		i++
	}
	/* update pool tx num */
	pool.pending = pool.pending[i:]
	return txs
}

func (pool *TxPool) Empty() bool {
	return len(pool.pending) == 0
}

func (pool *TxPool) PendingLen() int {
	return len(pool.pending)
}
