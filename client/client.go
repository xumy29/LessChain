package client

import (
	"container/list"
	"fmt"
	"go-w3chain/beaconChain"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
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
	/* 一笔cross1交易的信标被确认后，cross2交易的信标应该在rollbackHeight个区块高度内确认，否则客户端会发起回滚交易 */
	rollbackHeight int

	messageHub core.MessageHub

	addrTable map[common.Address]int

	stopCh chan struct{}

	shard_num int

	/* 待注入到分片的交易*/
	txs []*core.Transaction

	/* txs 中交易ID到交易的映射 */
	txs_map map[uint64]*core.Transaction

	/* 委员会发送给客户端的待处理的交易收据 */
	tx_reply *list.List
	/* tx_reply 的锁 */
	r_lock sync.Mutex

	/**
	 * 跨片交易前半部分在信标链上的确认高度，客户端接收到前半部分交易的收据后记录该map
	 * key：交易ID	value：确认高度，注意是tb.ConfirmHeight，不是tb.Height
	 * 使用记得加锁
	 */
	cross1_confirm_height_map map[uint64]uint64
	/* cross1_confirm_height_map 的锁 */
	c1_c_lock sync.RWMutex

	/* 待发送的跨片交易后半部分交易，由 cross1_tx_reply 中的交易在收到信标确认后转化过来。使用记得加锁*/
	cross2_txs []*core.Transaction
	/* cross2_txs的锁 */
	c2_lock sync.Mutex

	/* 跨片交易超时队列，客户端将向源分片发送回滚交易。使用记得加锁 */
	cross_tx_expired []uint64
	e_lock           sync.Mutex

	/* 本地存储的信标 */
	tbs            map[int]map[uint64]*beaconChain.TimeBeacon
	tbchain_height uint64

	wg sync.WaitGroup
}

func NewClient(id, rollbackHeight, shardNum int) *Client {
	c := &Client{
		ip:                        defaultIP,
		port:                      defaultPort,
		cid:                       id,
		rollbackHeight:            rollbackHeight,
		stopCh:                    make(chan struct{}),
		txs:                       make([]*core.Transaction, 0),
		txs_map:                   make(map[uint64]*core.Transaction),
		tx_reply:                  list.New(),
		shard_num:                 shardNum,
		cross1_confirm_height_map: make(map[uint64]uint64),
		tbs:                       make(map[int]map[uint64]*beaconChain.TimeBeacon),
	}

	for shardID := 0; shardID < shardNum; shardID++ {
		c.tbs[shardID] = make(map[uint64]*beaconChain.TimeBeacon)
	}
	return c
}

func (c *Client) Start(injectSpeed, recommitIntervalSecs int, addrTable map[common.Address]int) {
	c.addrTable = addrTable
	c.wg.Add(1)
	go c.SendTXs(injectSpeed, addrTable)
	// c.wg.Add(1)
	// go c.CheckExpiredTXs(recommitIntervalSecs)
}

func (c *Client) SetMessageHub(hub core.MessageHub) {
	c.messageHub = hub
}

/* 检查是否有超时的跨片交易 */
func (c *Client) checkExpiredTXs() {
	now := time.Now().Unix()
	c.c1_c_lock.Lock()
	expired_txs := make([]uint64, 0, len(c.cross1_confirm_height_map)/2)
	for txid, confirmHeight := range c.cross1_confirm_height_map {
		if c.tbchain_height >= confirmHeight+c.txs_map[txid].RollbackHeight {
			// log.Debug("checkExpiredTX", "txid", txid, "tbchain_height", c.tbchain_height,
			// 	"confirmHeight", confirmHeight, "rollbackHeight", c.txs_map[txid].RollbackHeight)

			// 发送回滚交易前应判断交易后半部分是否已被打包，若已被打包则不发送回滚交易
			tx := c.txs_map[txid]
			cross2_packed := c.checkCross2TxPacked(tx)
			if cross2_packed {
				log.Trace("tracing transaction", "txid", tx.ID, "status", result.GetStatusString(result.RollbackFail), "time", now)
			} else {
				expired_txs = append(expired_txs, txid)
				log.Trace("tracing transaction", "txid", txid, "status", "client add tx to expired_tx", "time", now)
			}
			delete(c.cross1_confirm_height_map, txid)
		}
	}
	c.c1_c_lock.Unlock()
	c.e_lock.Lock()
	defer c.e_lock.Unlock()
	c.cross_tx_expired = append(c.cross_tx_expired, expired_txs...)
	// log.Debug("checkExpiredTXs", "cid", c.cid, "cross1_confirm_height_map_len", len(c.cross1_confirm_height_map), "expired_tx_len", len(expired_txs))

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

/* 所有交易执行完成则结束 */
func (c *Client) CanStopV1() bool {
	log.Trace("client queue length", "tx_reply", c.tx_reply.Len(), "cross_tx_expired", len(c.cross_tx_expired),
		"cross2_txs", len(c.cross2_txs), "cross1_confirm_height_map", len(c.cross1_confirm_height_map))
	return c.CanStopV2() && c.tx_reply.Len() == 0 && len(c.cross_tx_expired) == 0 &&
		len(c.cross2_txs) == 0 && len(c.cross1_confirm_height_map) == 0
}

/* 客户端初始交易注入完成则结束 */
func (c *Client) CanStopV2() bool {
	return true
}
