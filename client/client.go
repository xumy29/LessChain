package client

import (
	"fmt"
	"go-w3chain/beaconChain"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"go-w3chain/shard"
	"go-w3chain/utils"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var (
	defaultIP   string
	defaultPort int
)

type Client struct {
	ip   string
	port int

	cid int
	/* 收到一笔cross1交易的回复后，cross2交易应该在rollbackSec内commit，否则客户端会发起回滚交易 */
	rollbackSec int

	messageHub core.MessageHub

	stopCh chan struct{}

	cross1_reply_time_map map[uint64]uint64
	shard_num             int

	/* 待注入到分片的交易 */
	txs  []*core.Transaction
	lock sync.Mutex
	/* 交易ID到交易的映射 */
	txs_map map[uint64]*core.Transaction

	/* 分片回复给客户端的消息，只有跨片交易前半部分回复消息会被加到此队列，等待处理 */
	cross1_tx_reply []*result.TXReceipt
	c1_lock         sync.Mutex
	/* 跨片交易超时队列，客户端将向源分片发送回滚交易 */
	cross_tx_expired []uint64
	e_lock           sync.Mutex

	tbs map[uint64]map[uint64]*beaconChain.TimeBeacon
}

func NewClient(id, rollbackSecs, shardNum int) *Client {
	c := &Client{
		ip:                    defaultIP,
		port:                  defaultPort,
		cid:                   id,
		rollbackSec:           rollbackSecs,
		stopCh:                make(chan struct{}),
		txs:                   make([]*core.Transaction, 0),
		txs_map:               make(map[uint64]*core.Transaction),
		shard_num:             shardNum,
		cross1_tx_reply:       make([]*result.TXReceipt, 0),
		cross1_reply_time_map: make(map[uint64]uint64),
		tbs:                   make(map[uint64]map[uint64]*beaconChain.TimeBeacon),
	}
	return c
}

func (c *Client) SetMessageHub(hub core.MessageHub) {
	c.messageHub = hub
}

/**
 * 该函数在loadData后被调用一次，将分配到此客户端的交易加入队列中，并多设一个map用于快速根据交易ID找到交易
 */
func (c *Client) Addtxs(txs []*core.Transaction) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.txs = append(c.txs, txs...)
	// log.Debug("clientAddtxs", "clientID", c.cid, "txs_len", len(c.txs))
	for _, tx := range c.txs {
		c.txs_map[tx.ID] = tx
	}
}

/**
 * 按一定速率发送交易到分片
 * 目前的实现未通过网络传输
 */
func (c *Client) SendTXs(inject_speed int, shards []*shard.Shard, addrTable map[common.Address]int) {
	c.InjectTXs(c.cid, inject_speed, addrTable)
}

/* 接收分片执行完交易的收据 */
func (c *Client) AddTXReceipts(receipts []*result.TXReceipt) {
	c.c1_lock.Lock()
	defer c.c1_lock.Unlock()
	cross1Cnt := 0
	cross2Cnt := 0

	if len(receipts) == 0 {
		return
	}
	// 获取对应分片和高度的信标
	tb := c.GetTB(uint64(receipts[0].ShardID), receipts[0].BlockHeight)
	log.Debug("ClientGetTimeBeacon", "info", tb)

	for _, r := range receipts {
		validity := VerifyTxMKproof(r.MKproof, tb)
		if !validity {
			log.Error("transaction's merkle proof verification didn't pass!")
		}
		if r.TxStatus == result.CrossTXType1Success {
			cross1Cnt += 1
			c.cross1_tx_reply = append(c.cross1_tx_reply, r)
			if r.ConfirmTimeStamp == 0 {
				log.Warn("ConfirmTimeStamp == 0! confirmTimeStamp is expected greater than zero", "txid", r.TxID, "txstatus", r.TxStatus)
			}
			c.cross1_reply_time_map[r.TxID] = r.ConfirmTimeStamp

		} else if r.TxStatus == result.CrossTXType2Success {
			cross2Cnt += 1
			if cross1TxConfirmTime, ok := c.cross1_reply_time_map[r.TxID]; !ok {
				log.Warn("got cross2TXReply, but txid not in cross1_reply_time_map")
			} else {
				if r.ConfirmTimeStamp > cross1TxConfirmTime+uint64(c.rollbackSec) {
					log.Warn("got cross2Reply, but confirmTime exceeds rollback duration, recipient's shard may be wrong")
				} else {
					delete(c.cross1_reply_time_map, r.TxID)
				}
			}
		}
	}

	log.Debug("clientAddTXReceipt", "cid", c.cid, "receiptCnt", len(receipts), "cross1ReceiptCnt", cross1Cnt, "cross2ReceiptCnt", cross2Cnt, "cross1_tx_reply_queue_len", len(c.cross1_tx_reply))
}

/*
 * 客户端定期检查是否有超时的跨分片交易
 */
func (c *Client) CheckExpiredTXs(recommitIntervalSecs int) {
	recommitInterval := time.Duration(recommitIntervalSecs) * time.Second
	timer := time.NewTimer(recommitInterval)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			c.checkExpiredTXs()
			timer.Reset(recommitInterval)

		case <-c.stopCh:
			return
		}
	}
}

func (c *Client) checkExpiredTXs() {
	now := time.Now().Unix()
	expired_txs := make([]uint64, 0, len(c.cross1_reply_time_map)/2)
	for txid, confirmTime := range c.cross1_reply_time_map {
		extra_duration := 2 // 为了避免客户端将cross1视为超时后立刻收到分片关于cross2的回复
		if now > int64(confirmTime)+int64(c.rollbackSec)+int64(extra_duration) {
			delete(c.cross1_reply_time_map, txid)
			expired_txs = append(expired_txs, txid)
		}
	}
	c.e_lock.Lock()
	defer c.e_lock.Unlock()
	c.cross_tx_expired = append(c.cross_tx_expired, expired_txs...)
	log.Debug("checkExpiredTXs", "cid", c.cid, "cross1_reply_time_map_len", len(c.cross1_reply_time_map), "expired_tx_len", len(expired_txs))

}

func (c *Client) Stop() {
	close(c.stopCh)
}

func (c *Client) Print() {
	msg := fmt.Sprintf("client %d tx number: %d", c.cid, len(c.txs))
	log.Debug(msg)
}

/**
 * 按一定速率将客户端的交易注入到分片
 */
func (c *Client) InjectTXs(cid int, inject_speed int, addrTable map[common.Address]int) {
	cnt := 0
	resBroadcastMap := make(map[uint64]uint64)
	// 按秒注入
	for {
		time.Sleep(1000 * time.Millisecond) //fixme 应该记录下面的运行时间
		// start := time.Now().UnixMilli()
		rollbackTxSentCnt := c.sendRollbackTxs(inject_speed, addrTable)

		cross2TxSentCnt := c.sendCross2Txs(inject_speed-rollbackTxSentCnt, addrTable)

		cnt = c.sendPendingTxs(cnt, inject_speed-rollbackTxSentCnt-cross2TxSentCnt, addrTable, resBroadcastMap)
		if cnt == len(c.txs) {
			break
		}

	}
	/* 记录广播结果 */
	result.SetBroadcastMap(resBroadcastMap)

	/* 通知分片交易注入完成 */
	for i := 0; i < c.shard_num; i++ {
		c.messageHub.Send(core.MsgTypeSetInjectDone2Shard, uint64(i), struct{}{}, nil)
	}

}

func (c *Client) sendRollbackTxs(maxTxNum2Pack int, addrTable map[common.Address]int) int {
	c.e_lock.Lock()
	defer c.e_lock.Unlock()
	rollbackTxSentCnt := utils.Min(maxTxNum2Pack, len(c.cross_tx_expired))
	if rollbackTxSentCnt == 0 {
		return 0
	}

	/* 初始化 shardtxs */
	shardtxs := make([][]*core.Transaction, c.shard_num)
	for i := 0; i < c.shard_num; i++ {
		shardtxs[i] = make([]*core.Transaction, 0, rollbackTxSentCnt*2/c.shard_num+1)
	}
	for i := 0; i < rollbackTxSentCnt; i++ {
		txid := c.cross_tx_expired[i]
		tx := *c.txs_map[txid]
		tx.TXtype = core.RollbackTXType
		tx.TXStatus = result.DefaultStatus
		shardidx := addrTable[*tx.Sender]
		shardtxs[shardidx] = append(shardtxs[shardidx], &tx)
	}
	/* 注入到各分片 */
	for i := 0; i < c.shard_num; i++ {
		c.messageHub.Send(core.MsgTypeClientInjectTX2Committee, uint64(i), shardtxs[i], nil)
	}
	// 移除已发送的reply交易
	c.cross_tx_expired = c.cross_tx_expired[rollbackTxSentCnt:]
	return rollbackTxSentCnt
}

/* 从cross1回复队列中取交易发送对应的cross2交易，优先于pending队列, 交易被发送后从队列中移除*/
func (c *Client) sendCross2Txs(maxTxNum2Pack int, addrTable map[common.Address]int) int {
	c.c1_lock.Lock()
	defer c.c1_lock.Unlock()
	cross2TxSentCnt := utils.Min(maxTxNum2Pack, len(c.cross1_tx_reply))
	if cross2TxSentCnt == 0 {
		return 0
	}

	/* 初始化 shardtxs */
	shardtxs := make([][]*core.Transaction, c.shard_num)
	for i := 0; i < c.shard_num; i++ {
		shardtxs[i] = make([]*core.Transaction, 0, cross2TxSentCnt*2/c.shard_num+1)
	}
	for i := 0; i < cross2TxSentCnt; i++ {
		txid := c.cross1_tx_reply[i].TxID
		tx := *c.txs_map[txid]
		tx.TXtype = core.CrossTXType2
		tx.TXStatus = result.DefaultStatus
		tx.RollbackSecs = uint64(c.rollbackSec)
		shardidx := addrTable[*tx.Recipient]
		shardtxs[shardidx] = append(shardtxs[shardidx], &tx)
	}
	/* 注入到各分片 */
	for i := 0; i < c.shard_num; i++ {
		c.messageHub.Send(core.MsgTypeClientInjectTX2Committee, uint64(i), shardtxs[i], nil)
	}
	// 移除已发送的reply交易
	c.cross1_tx_reply = c.cross1_tx_reply[cross2TxSentCnt:]
	return cross2TxSentCnt
}

/* 从pending队列中取交易发送 */
func (c *Client) sendPendingTxs(cnt, maxTxNum2Pack int, addrTable map[common.Address]int, resBroadcastMap map[uint64]uint64) int {
	if maxTxNum2Pack == 0 {
		return cnt
	}

	upperBound := utils.Min(cnt+maxTxNum2Pack, len(c.txs))
	/* 初始化 shardtxs */
	shardtxs := make([][]*core.Transaction, c.shard_num)
	for i := 0; i < c.shard_num; i++ {
		shardtxs[i] = make([]*core.Transaction, 0, maxTxNum2Pack*2/c.shard_num+1)
	}

	for i := cnt; i < upperBound; i++ {
		tx := c.txs[i]
		// 根据发送地址和接收地址确认交易类型
		if addrTable[*tx.Sender] == addrTable[*tx.Recipient] {
			tx.TXtype = core.IntraTXType
		} else {
			tx.TXtype = core.CrossTXType1
		}

		tx.Timestamp = uint64(time.Now().Unix())
		tx.Cid = uint64(c.cid)
		resBroadcastMap[tx.ID] = tx.Timestamp
		shardidx := addrTable[*tx.Sender]
		shardtxs[shardidx] = append(shardtxs[shardidx], tx)

	}
	/* 注入到各分片 */
	for i := 0; i < c.shard_num; i++ {
		c.messageHub.Send(core.MsgTypeClientInjectTX2Committee, uint64(i), shardtxs[i], nil)
	}
	/* 更新循环变量 */
	cnt = upperBound
	return cnt
}

func (c *Client) getTBFromTBChain(shardID, height uint64) *beaconChain.TimeBeacon {
	var tb *beaconChain.TimeBeacon
	callback := func(res ...interface{}) {
		tb = res[0].(*beaconChain.TimeBeacon)
	}
	c.messageHub.Send(core.MsgTypeGetTB, shardID, height, callback)
	return tb
}

func (c *Client) GetTB(shardID, height uint64) *beaconChain.TimeBeacon {
	if _, ok := c.tbs[shardID]; !ok {
		c.tbs[shardID] = make(map[uint64]*beaconChain.TimeBeacon)
	}
	if tb, ok := c.tbs[shardID][height]; !ok {
		tb1 := c.getTBFromTBChain(shardID, height)
		c.tbs[shardID][height] = tb1
		return tb1
	} else {
		return tb
	}
}

/* 验证一笔交易的merkle proof，tb 是该交易对应分片和高度的区块信标
* 目前未实现具体逻辑，直接假设验证通过
 */
func VerifyTxMKproof(proof []byte, tb *beaconChain.TimeBeacon) bool {
	return true
}

// func (c *Client) GetTXs() []*core.Transaction {
// 	return c.txs
// }

// func (c *Client) GetCross1TXReply() []*result.TXReceipt {
// 	return c.cross1_tx_reply
// }

// func (c *Client) SetCross1TXReply(new_txs_reply []*result.TXReceipt) {
// 	c.c1_lock.Lock()
// 	defer c.c1_lock.Unlock()
// 	old_len := len(c.cross1_tx_reply)
// 	c.cross1_tx_reply = new_txs_reply
// 	log.Debug("SetTXReceipts", "old_txs_reply_len", old_len, "txs_reply_len", len(c.cross1_tx_reply))
// }

// func (c *Client) GetRollbackSecs() uint64 {
// 	return uint64(c.rollbackSec)
// }
