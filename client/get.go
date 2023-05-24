package client

import (
	"go-w3chain/beaconChain"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
)

/* 接收分片执行完交易的收据 */
func (c *Client) AddTXReceipts(receipts []*result.TXReceipt) {
	c.c1_lock.Lock()
	defer c.c1_lock.Unlock()
	c.c1_c_lock.Lock()
	defer c.c1_c_lock.Unlock()
	cross1Cnt := 0
	cross2Cnt := 0

	if len(receipts) == 0 {
		return
	}
	// // 获取对应分片和高度的信标
	// tb := c.GetTB(receipts[0].ShardID, receipts[0].BlockHeight)
	// log.Debug("ClientGetTimeBeacon", "info", tb)

	for _, r := range receipts {
		// validity := VerifyTxMKproof(r.MKproof, tb)
		// if !validity {
		// 	log.Error("transaction's merkle proof verification didn't pass!")
		// }
		if r.TxStatus == result.CrossTXType1Success {
			cross1Cnt += 1
			c.cross1_tx_reply.PushBack(r)
			if r.ConfirmTimeStamp == 0 {
				log.Warn("ConfirmTimeStamp == 0! confirmTimeStamp is expected greater than zero", "txid", r.TxID, "txstatus", r.TxStatus)
			}
			c.cross1_confirm_time_map[r.TxID] = r.ConfirmTimeStamp

		} else if r.TxStatus == result.CrossTXType2Success {
			cross2Cnt += 1
			if cross1TxConfirmTime, ok := c.cross1_confirm_time_map[r.TxID]; !ok {
				log.Warn("got cross2TXReply, but txid not in cross1_confirm_time_map")
			} else {
				if r.ConfirmTimeStamp > cross1TxConfirmTime+uint64(c.rollbackSec) {
					log.Warn("got cross2Reply, but confirmTime exceeds rollback duration, recipient's shard may be wrong")
				} else {
					delete(c.cross1_confirm_time_map, r.TxID)
				}
			}
		}
	}

	log.Debug("clientAddTXReceipt", "cid", c.cid, "receiptCnt", len(receipts),
		"cross1ReceiptCnt", cross1Cnt, "cross2ReceiptCnt", cross2Cnt,
		"cross1_tx_reply_queue_len", c.cross1_tx_reply.Len())
}

/** 获取信标
 * 先从本地缓存拿，如果本地没有再从信标链拿
 */
func (c *Client) GetTB(shardID int, height uint64) *beaconChain.TimeBeacon {
	if tb, ok := c.tbs[shardID][height]; !ok {
		tb1 := c.getTBFromTBChain(shardID, height)
		c.tbs[shardID][height] = tb1
		return tb1
	} else {
		return tb
	}
}

/** 信标链主动向客户端推送新确认的信标时调用此函数
* 一般情况下信标链应该只向客户端推送其关注的分片和高度的信标，这里进行了简化，默认全部推送
* 客户端收到新确认信标后，遍历 cross1_tx_reply，如果某个交易的区块信标已确认，
将该交易加到 cross2_txs 中，可以作为跨片交易后半部分发送给委员会
*/
func (c *Client) AddTBs(tbs_new map[int][]*beaconChain.TimeBeacon) {
	for shardID, tbs := range tbs_new {
		for _, tb := range tbs {
			c.tbs[shardID][tb.Height] = tb
		}
	}
	c.c1_lock.Lock()
	defer c.c1_lock.Unlock()
	c.c2_lock.Lock()
	defer c.c2_lock.Unlock()
	l := c.cross1_tx_reply
	for e := l.Front(); e != nil; {
		n := e.Next()

		r := e.Value.(*result.TXReceipt)
		if _, ok := c.tbs[r.ShardID][r.BlockHeight]; ok { // 该交易的信标已被确认
			txid := r.TxID
			tx := *c.txs_map[txid]
			tx.TXtype = core.CrossTXType2
			tx.TXStatus = result.DefaultStatus
			tx.RollbackSecs = uint64(c.rollbackSec)
			c.cross2_txs = append(c.cross2_txs, &tx)
			l.Remove(e)
		}

		e = n
	}

}
