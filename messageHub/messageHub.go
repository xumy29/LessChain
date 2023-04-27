package messageHub

import (
	"go-w3chain/client"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"go-w3chain/shard"
)

type GoodMessageHub struct {
	mid int
}

/* 是controller中的shards和clients的引用 */
var shards_ref []*shard.Shard
var clients_ref []*client.Client

/* 用于分片和客户端传送消息 */
func (hub *GoodMessageHub) Send(msgType uint64, toid int, msg interface{}) {
	switch msgType {
	case core.MsgTypeShardReply2Client:
		client := clients_ref[toid]
		receipts := msg.([]*result.TXReceipt)
		client.AddTXReceipts(receipts)

	case core.MsgTypeClientInjectTX2Shard:
		shard := shards_ref[toid]
		txs := msg.([]*core.Transaction)
		shard.InjectTXs(txs)

	case core.MsgTypeSetInjectDone2Shard:
		shard := shards_ref[toid]
		shard.SetInjectTXDone()

	}
}

func Init(clients []*client.Client, shards []*shard.Shard) {
	clients_ref = clients
	shards_ref = shards
	log.Info("messageHubInit", "clientNum", len(clients_ref), "shardNum", len(shards_ref))
	hub := &GoodMessageHub{
		mid: 1,
	}
	for _, c := range clients_ref {
		c.SetMessageHub(hub)
	}
	for _, s := range shards_ref {
		s.SetMessageHub(hub)
	}
}
