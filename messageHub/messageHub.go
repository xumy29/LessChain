package messageHub

import (
	"go-w3chain/beaconChain"
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
var tbChain_ref *beaconChain.BeaconChain

/* 用于分片、客户端、信标链传送消息 */
func (hub *GoodMessageHub) Send(msgType uint64, id uint64, msg interface{}, callback func(res interface{})) {
	switch msgType {
	case core.MsgTypeShardReply2Client:
		client := clients_ref[id]
		receipts := msg.([]*result.TXReceipt)
		client.AddTXReceipts(receipts)

	case core.MsgTypeClientInjectTX2Shard:
		shard := shards_ref[id]
		txs := msg.([]*core.Transaction)
		shard.InjectTXs(txs)

	case core.MsgTypeSetInjectDone2Shard:
		shard := shards_ref[id]
		shard.SetInjectTXDone()

	case core.MsgTypeAddTB:
		tb := msg.(*beaconChain.TimeBeacon)
		tbChain_ref.AddTimeBeacon(tb)

	case core.MsgTypeGetTB:
		height := msg.(uint64)
		tb := tbChain_ref.GetTimeBeacon(id, height)
		callback(tb)
	}
}

func Init(clients []*client.Client, shards []*shard.Shard, tbChain *beaconChain.BeaconChain) {
	clients_ref = clients
	shards_ref = shards
	tbChain_ref = tbChain
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
	tbChain_ref.SetMessageHub(hub)
}
