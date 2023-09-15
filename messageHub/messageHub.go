package messageHub

import (
	"encoding/gob"
	"go-w3chain/beaconChain"
	"go-w3chain/client"
	"go-w3chain/committee"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/node"
	"go-w3chain/shard"
	"net"
	"sync"
)

var (
	shard_ref     *shard.Shard
	committee_ref *committee.Committee
	client_ref    *client.Client
	node_ref      *node.Node
	booter_ref    *node.Booter
	tbChain_ref   *beaconChain.BeaconChain

	shardNum     int
	clientNum    int
	conns2Shard  *ConnectionsMap
	conns2Com    *ConnectionsMap
	conns2Client *ConnectionsMap
	listenConn   net.Listener
)

func init() {
	conns2Shard = NewConnectionsMap()
	conns2Com = NewConnectionsMap()
	conns2Client = NewConnectionsMap()
	gob.Register(core.Msg{})
	gob.Register(core.BooterSendContract{})

}

type GoodMessageHub struct {
	mid      int
	exitChan chan struct{}
}

func NewMessageHub() *GoodMessageHub {
	hub := &GoodMessageHub{
		mid:      1,
		exitChan: make(chan struct{}, 1),
	}
	return hub
}

func (hub *GoodMessageHub) Init(client *client.Client, node *node.Node, booter *node.Booter,
	tbChain *beaconChain.BeaconChain, _shardNum int, _clientNum int, wg *sync.WaitGroup) {
	clientNum = _clientNum
	client_ref = client

	node_ref = node
	if node_ref != nil {
		shard_ref = node.GetShard().(*shard.Shard)
		committee_ref = node.GetCommittee().(*committee.Committee)
	}

	booter_ref = booter
	tbChain_ref = tbChain
	shardNum = _shardNum
	log.Info("messageHubInit", "shardNum", shardNum)

	if committee_ref != nil {
		committee_ref.SetMessageHub(hub)
	}
	if shard_ref != nil {
		shard_ref.SetMessageHub(hub)
	}
	if client_ref != nil {
		client_ref.SetMessageHub(hub)
		wg.Add(1)
		go listen(client.GetAddr(), wg)
	}
	// 目前所有角色都需要连接tbchain
	if tbChain_ref == nil {
		log.Error("tbchain instance is nil")
	}
	tbChain_ref.SetMessageHub(hub)

	if node_ref != nil {
		wg.Add(1)
		go listen(node_ref.GetAddr(), wg)
	}
	if booter_ref != nil {
		booter_ref.SetMessageHub(hub)
		wg.Add(1)
		go listen(booter.GetAddr(), wg)
	}

}

func (hub GoodMessageHub) Close() {
	// 关闭所有tcp连接，防止资源泄露
	for _, conn := range conns2Client.connections {
		conn.Close()
	}
	for _, conn := range conns2Com.connections {
		conn.Close()
	}
	for _, conn := range conns2Shard.connections {
		conn.Close()
	}
	listenConn.Close()
}

/* 用于分片、委员会、客户端、信标链传送消息 */
func (hub *GoodMessageHub) Send(msgType uint32, id uint32, msg interface{}, callback func(res ...interface{})) {
	switch msgType {
	case core.MsgTypeComGetHeightFromShard:
		height := comGetHeightFromShard(id, msg)
		callback(height)

	case core.MsgTypeShardSendGenesis:
		shardSendGenesis(msg)
	case core.MsgTypeBooterSendContract:
		booterSendContract(msg)

	case core.MsgTypeComGetStateFromShard:
		comGetStateFromShard(id, msg)
	case core.MsgTypeShardSendStateToCom:
		go shardSendStateToCom(id, msg)

	case core.MsgTypeClientInjectTX2Committee:
		go clientInjectTx2Com(id, msg)
	case core.MsgTypeSetInjectDone2Nodes:
		clientSetInjectDone2Nodes(id)

	case core.MsgTypeSendBlock2Shard:
		go comSendBlock2Shard(id, msg)

	case core.MsgTypeCommitteeReply2Client:
		go comSendReply2Client(id, msg)

	case core.MsgTypeComAddTb2TBChain:
		comAddTb2TBChain(msg)

	////////////////////
	// 通过beaconChain模块中的ethclient与ethChain交互
	///////////////////
	case core.MsgTypeComGetLatestBlockHashFromEthChain:
		comGetLatestBlock(id, callback)

	case core.MsgTypeTBChainPushTB2Client:
		tbChainPushBlock2Client(msg)

		// client
		// case core.MsgTypeClientInjectTX2Committee:
		// 	clientInjectTx2Com(id, msg)
		// case core.MsgTypeCommitteeReply2Client:
		// 	client := clients_ref[id]
		// 	receipts := msg.([]*result.TXReceipt)
		// 	client.AddTXReceipts(receipts)
		// 	// client -> committee
		// case core.MsgTypeSetInjectDone2Nodes:
		// 	com := committees_ref[id]
		// 	com.SetInjectTXDone()
		// 	// shard、committee -> tbchain
		// case core.MsgTypeComAddTb2TBChain:
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
		// case core.MsgTypeSendBlock2Shard:
		// 	shard := shards_ref[id]
		// 	block := msg.(*core.Block)
		// 	shard.Addblock(block)
		// 	// committee -> hub
		// case core.MsgTypeReady4Reconfig:
		// 	seedHeight := msg.(uint64)
		// 	toReconfig(seedHeight)
		// case core.MsgTypeTBChainPushTB2Client:
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
