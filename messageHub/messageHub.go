package messageHub

import (
	"go-w3chain/beaconChain"
	"go-w3chain/client"
	"go-w3chain/committee"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/node"
	"go-w3chain/shard"
)

var (
	shard_ref     *shard.Shard
	committee_ref *committee.Committee
	client_ref    *client.Client
	node_ref      *node.Node
	tbChain_ref   *beaconChain.BeaconChain

	conns2Shard  *ConnectionsMap
	conns2Com    *ConnectionsMap
	conns2Client *ConnectionsMap
)

func init() {
	conns2Shard = NewConnectionsMap()
	conns2Com = NewConnectionsMap()
	conns2Client = NewConnectionsMap()
}

type GoodMessageHub struct {
	mid int
}

func NewMessageHub() *GoodMessageHub {
	hub := &GoodMessageHub{
		mid: 1,
	}
	return hub
}

func (hub *GoodMessageHub) Init(client *client.Client, shard *shard.Shard,
	committee *committee.Committee, node *node.Node,
	tbChain *beaconChain.BeaconChain, shardNum int) {
	client_ref = client
	shard_ref = shard
	committee_ref = committee
	node_ref = node
	tbChain_ref = tbChain
	log.Info("messageHubInit", "shardNum", shardNum)

	if committee_ref != nil {
		committee_ref.SetMessageHub(hub)
	}
	if shard_ref != nil {
		shard_ref.SetMessageHub(hub)
	}
	if client_ref != nil {
		client_ref.SetMessageHub(hub)
	}
	if tbChain_ref == nil {
		log.Error("tbchain instance is nil")
	}
	tbChain_ref.SetMessageHub(hub)

	if node_ref != nil {
		listen(node_ref.GetAddr())
	}

}

func (hub GoodMessageHub) Close() {
	// 关闭所有tcp连接，防止资源泄露
	for _, conn := range conns2Client.connections {
		conn.Close()
	}
}

/* 用于分片、委员会、客户端、信标链传送消息 */
func (hub *GoodMessageHub) Send(msgType uint32, id uint32, msg interface{}, callback func(res ...interface{})) {
	switch msgType {
	case core.MsgTypeComGetStateFromShard:
		comGetStateFromShard(id, msg)
	case core.MsgTypeShardSendStateToCom:
		shardSendStateToCom(id, msg)
		// client
		// case core.MsgTypeClientInjectTX2Committee:
		// 	clientInjectTx2Com(id, msg)
		// case core.MsgTypeCommitteeReply2Client:
		// 	client := clients_ref[id]
		// 	receipts := msg.([]*result.TXReceipt)
		// 	client.AddTXReceipts(receipts)
		// 	// client -> committee
		// case core.MsgTypeSetInjectDone2Committee:
		// 	com := committees_ref[id]
		// 	com.SetInjectTXDone()
		// 	// shard、committee -> tbchain
		// case core.MsgTypeCommitteeAddTB:
		// 	tb := msg.(*beaconChain.SignedTB)
		// 	tbChain_ref.AddTimeBeacon(tb)
		// case core.MsgTypeCommitteeInitialAddrs:
		// 	addrs := msg.([]common.Address)
		// 	tbChain_ref.SetAddrs(addrs, nil, 0, uint32(id))
		// case core.MsgTypeCommitteeAdjustAddrs:
		// 	data := msg.(*committee.AdjustAddrs)
		// 	tbChain_ref.SetAddrs(data.Addrs, data.Vrfs, data.SeedHeight, uint32(id))
		// 	// committee、client <- tbchain
		// case core.MsgTypeGetTB:
		// 	height := msg.(uint64)
		// 	tb := tbChain_ref.GetTimeBeacon(int(id), height)
		// 	callback(tb)
		// 	// committee <- shard
		// case core.MsgTypeComGetStateFromShard:
		// 	shard := shards_ref[id]
		// 	states := shard.GetBlockChain().GetStateDB()
		// parentHeight := shard.GetBlockChain().CurrentBlock().Number()
		// 	callback(states, parentHeight)
		// 	// committee -> shard
		// case core.MsgTypeAddBlock2Shard:
		// 	shard := shards_ref[id]
		// 	block := msg.(*core.Block)
		// 	shard.Addblock(block)
		// 	// committee -> hub
		// case core.MsgTypeReady4Reconfig:
		// 	seedHeight := msg.(uint64)
		// 	toReconfig(seedHeight)
		// case core.MsgTypeTBChainPushTB2Clients:
		// 	block := msg.(*beaconChain.TBBlock)
		// 	for _, c := range clients_ref {
		// 		c.AddTBs(block)
		// 	}
		// case core.MsgTypeTBChainPushTB2Coms:
		// 	block := msg.(*beaconChain.TBBlock)
		// 	for _, c := range committees_ref {
		// 		c.AddTBs(block)
		// 	}
	}
}

func (hub *GoodMessageHub) Receive(msgType uint32, id uint32, msg interface{}, callback func(res ...interface{})) {
	switch msgType {
	// committee
	case core.MsgTypeClientInjectTX2Committee:
		com := committee_ref
		txs := msg.([]*core.Transaction)
		com.InjectTXs(txs)
		// committee -> client
		// case core.MsgTypeCommitteeReply2Client:
		// 	client := clients_ref[id]
		// 	receipts := msg.([]*result.TXReceipt)
		// 	client.AddTXReceipts(receipts)
		// 	// client -> committee
		// case core.MsgTypeSetInjectDone2Committee:
		// 	com := committees_ref[id]
		// 	com.SetInjectTXDone()
		// 	// shard、committee -> tbchain
		// case core.MsgTypeCommitteeAddTB:
		// 	tb := msg.(*beaconChain.SignedTB)
		// 	tbChain_ref.AddTimeBeacon(tb)
		// case core.MsgTypeCommitteeInitialAddrs:
		// 	addrs := msg.([]common.Address)
		// 	tbChain_ref.SetAddrs(addrs, nil, 0, uint32(id))
		// case core.MsgTypeCommitteeAdjustAddrs:
		// 	data := msg.(*committee.AdjustAddrs)
		// 	tbChain_ref.SetAddrs(data.Addrs, data.Vrfs, data.SeedHeight, uint32(id))
		// 	// committee、client <- tbchain
		// case core.MsgTypeGetTB:
		// 	height := msg.(uint64)
		// 	tb := tbChain_ref.GetTimeBeacon(int(id), height)
		// 	callback(tb)
		// 	// committee <- shard
		// case core.MsgTypeComGetStateFromShard:
		// 	shard := shards_ref[id]
		// 	states := shard.GetBlockChain().GetStateDB()
		// 	parentHeight := shard.GetBlockChain().CurrentBlock().Number()
		// 	callback(states, parentHeight)
		// 	// committee -> shard
		// case core.MsgTypeAddBlock2Shard:
		// 	shard := shards_ref[id]
		// 	block := msg.(*core.Block)
		// 	shard.Addblock(block)
		// 	// committee -> hub
		// case core.MsgTypeReady4Reconfig:
		// 	seedHeight := msg.(uint64)
		// 	toReconfig(seedHeight)
		// case core.MsgTypeTBChainPushTB2Clients:
		// 	block := msg.(*beaconChain.TBBlock)
		// 	for _, c := range clients_ref {
		// 		c.AddTBs(block)
		// 	}
		// case core.MsgTypeTBChainPushTB2Coms:
		// 	block := msg.(*beaconChain.TBBlock)
		// 	for _, c := range committees_ref {
		// 		c.AddTBs(block)
		// 	}
	}
}
