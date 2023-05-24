package client

import (
	"container/list"
	"fmt"
	"go-w3chain/beaconChain"
	"go-w3chain/core"
	"go-w3chain/log"
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

	shard_num int

	/* 待注入到分片的交易*/
	txs []*core.Transaction

	/* txs 中交易ID到交易的映射 */
	txs_map map[uint64]*core.Transaction

	/** 分片回复给客户端的消息，只有跨片交易前半部分回复消息会被加到此队列，等待处理。
	 * 使用记得加锁
	 * 列表元素类型是 *result.Receipt
	 */
	cross1_tx_reply *list.List
	/* cross1_tx_reply 的锁 */
	c1_lock sync.Mutex

	/**
	 * 跨片交易前半部分的确认时间，客户端接收到前半部分交易的收据后记录该map
	 * key：交易ID	value：确认时间
	 * 使用记得加锁
	 */
	cross1_confirm_time_map map[uint64]uint64
	/* cross1_confirm_time_map 的锁 */
	c1_c_lock sync.RWMutex

	/* 待发送的跨片交易后半部分交易，由 cross1_tx_reply 中的交易在收到信标确认后转化过来。使用记得加锁*/
	cross2_txs []*core.Transaction
	c2_lock    sync.Mutex

	/* 跨片交易超时队列，客户端将向源分片发送回滚交易。使用记得加锁 */
	cross_tx_expired []uint64
	e_lock           sync.Mutex

	/* 本地存储的信标 */
	tbs map[int]map[uint64]*beaconChain.TimeBeacon

	wg sync.WaitGroup
}

func NewClient(id, rollbackSecs, shardNum int) *Client {
	c := &Client{
		ip:                      defaultIP,
		port:                    defaultPort,
		cid:                     id,
		rollbackSec:             rollbackSecs,
		stopCh:                  make(chan struct{}),
		txs:                     make([]*core.Transaction, 0),
		txs_map:                 make(map[uint64]*core.Transaction),
		shard_num:               shardNum,
		cross1_tx_reply:         list.New(),
		cross1_confirm_time_map: make(map[uint64]uint64),
		tbs:                     make(map[int]map[uint64]*beaconChain.TimeBeacon),
	}

	for shardID := 0; shardID < shardNum; shardID++ {
		c.tbs[shardID] = make(map[uint64]*beaconChain.TimeBeacon)
	}
	return c
}

func (c *Client) Start(injectSpeed, recommitIntervalSecs int, addrTable map[common.Address]int) {
	c.wg.Add(1)
	go c.SendTXs(injectSpeed, addrTable)
	c.wg.Add(1)
	go c.CheckExpiredTXs(recommitIntervalSecs)
}

func (c *Client) SetMessageHub(hub core.MessageHub) {
	c.messageHub = hub
}

/**
 * 该函数在loadData后被调用一次，将分配到此客户端的交易加入队列中，并多设一个map用于快速根据交易ID找到交易
 */
func (c *Client) Addtxs(txs []*core.Transaction) {
	c.txs = append(c.txs, txs...)
	// log.Debug("clientAddtxs", "clientID", c.cid, "txs_len", len(c.txs))
	for _, tx := range c.txs {
		c.txs_map[tx.ID] = tx
	}
}

/*
 * 客户端定期检查是否有超时的跨分片交易
 */
func (c *Client) CheckExpiredTXs(recommitIntervalSecs int) {
	defer c.wg.Done()
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
	c.c1_c_lock.Lock()
	now := time.Now().Unix()
	expired_txs := make([]uint64, 0, len(c.cross1_confirm_time_map)/2)
	for txid, confirmTime := range c.cross1_confirm_time_map {
		extra_duration := 2 // 为了避免客户端将cross1视为超时后立刻收到分片关于cross2的回复
		if now > int64(confirmTime)+int64(c.rollbackSec)+int64(extra_duration) {
			delete(c.cross1_confirm_time_map, txid)
			expired_txs = append(expired_txs, txid)
		}
	}
	c.c1_c_lock.Unlock()
	c.e_lock.Lock()
	defer c.e_lock.Unlock()
	c.cross_tx_expired = append(c.cross_tx_expired, expired_txs...)
	log.Debug("checkExpiredTXs", "cid", c.cid, "cross1_confirm_time_map_len", len(c.cross1_confirm_time_map), "expired_tx_len", len(expired_txs))

}

func (c *Client) Stop() {
	close(c.stopCh)
	c.wg.Wait()
	log.Info("client stop.", "clientID", c.cid)
}

func (c *Client) Print() {
	msg := fmt.Sprintf("client %d tx number: %d", c.cid, len(c.txs))
	log.Debug(msg)
}

func (c *Client) getTBFromTBChain(shardID int, height uint64) *beaconChain.TimeBeacon {
	var tb *beaconChain.TimeBeacon
	callback := func(res ...interface{}) {
		tb = res[0].(*beaconChain.TimeBeacon)
	}
	c.messageHub.Send(core.MsgTypeGetTB, uint64(shardID), height, callback)
	return tb
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
