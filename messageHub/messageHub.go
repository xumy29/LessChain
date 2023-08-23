package messageHub

import (
	"go-w3chain/beaconChain"
	"go-w3chain/client"
	"go-w3chain/committee"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"go-w3chain/shard"
	"go-w3chain/utils"

	"github.com/ethereum/go-ethereum/common"
)

type GoodMessageHub struct {
	mid int
}

var (
	/* 是controller中的shards和clients的引用 */
	shards_ref     []*shard.Shard
	committees_ref []*committee.Committee
	clients_ref    []*client.Client
	nodes_ref      []*core.Node
	tbChain_ref    *beaconChain.BeaconChain

	/* 初始等于委员会数量，一个委员会调用ToReconfig时该变量减一，减到0时触发重组 */
	isNotToReconfig int

	/* 所有节点的ID，按委员会的顺序排序 */
	allnodes_id []int
)

func NewMessageHub() *GoodMessageHub {
	hub := &GoodMessageHub{
		mid: 1,
	}
	return hub
}

/* 用于分片、委员会、客户端、信标链传送消息 */
func (hub *GoodMessageHub) Send(msgType uint64, id uint64, msg interface{}, callback func(res ...interface{})) {
	switch msgType {
	// client -> committee
	case core.MsgTypeClientInjectTX2Committee:
		com := committees_ref[id]
		txs := msg.([]*core.Transaction)
		com.InjectTXs(txs)
		// committee -> client
	case core.MsgTypeCommitteeReply2Client:
		client := clients_ref[id]
		receipts := msg.([]*result.TXReceipt)
		client.AddTXReceipts(receipts)
		// client -> committee
	case core.MsgTypeSetInjectDone2Committee:
		com := committees_ref[id]
		com.SetInjectTXDone()
		// shard、committee -> tbchain
	case core.MsgTypeCommitteeAddTB:
		tb := msg.(*beaconChain.SignedTB)
		tbChain_ref.AddTimeBeacon(tb)
	case core.MsgTypeCommitteeInitialAddrs:
		addrs := msg.([]common.Address)
		tbChain_ref.SetAddrs(addrs, nil, 0, uint32(id))
	case core.MsgTypeCommitteeAdjustAddrs:
		data := msg.(*committee.AdjustAddrs)
		tbChain_ref.SetAddrs(data.Addrs, data.Vrfs, data.SeedHeight, uint32(id))
		// committee、client <- tbchain
	case core.MsgTypeGetTB:
		height := msg.(uint64)
		tb := tbChain_ref.GetTimeBeacon(int(id), height)
		callback(tb)
		// committee <- shard
	case core.MsgTypeComGetStateFromShard:
		shard := shards_ref[id]
		states := shard.GetBlockChain().GetStateDB()
		parentHeight := shard.GetBlockChain().CurrentBlock().Number()
		callback(states, parentHeight)
		// committee -> shard
	case core.MsgTypeAddBlock2Shard:
		shard := shards_ref[id]
		block := msg.(*core.Block)
		shard.Addblock(block)
		// committee -> hub
	case core.MsgTypeReady4Reconfig:
		seedHeight := msg.(uint64)
		toReconfig(seedHeight)
	case core.MsgTypeTBChainPushTB2Clients:
		block := msg.(*beaconChain.TBBlock)
		for _, c := range clients_ref {
			c.AddTBs(block)
		}
	case core.MsgTypeTBChainPushTB2Coms:
		block := msg.(*beaconChain.TBBlock)
		for _, c := range committees_ref {
			c.AddTBs(block)
		}
	case core.MsgTypeClientGetCross2ProofFromShard:
		shard := shards_ref[id]
		tx := msg.(*core.Transaction)
		packed := shard.CheckCross2Packed(tx)
		callback(packed)
	}
}

func (hub *GoodMessageHub) Init(clients []*client.Client, shards []*shard.Shard, committees []*committee.Committee, nodes []*core.Node, tbChain *beaconChain.BeaconChain) {
	clients_ref = clients
	shards_ref = shards
	committees_ref = committees
	nodes_ref = nodes
	isNotToReconfig = len(committees_ref)
	tbChain_ref = tbChain
	log.Info("messageHubInit", "clientNum", len(clients_ref), "shardNum", len(shards_ref))

	for _, c := range clients_ref {
		c.SetMessageHub(hub)
	}
	for _, s := range shards_ref {
		s.SetMessageHub(hub)
	}

	for _, c := range committees_ref {
		c.SetMessageHub(hub)
	}

	// 根据committee的初始节点初始化 allnodes_id，之后重组时打乱 allnodes_id，并根据结果得到委员会的新节点
	allnodes_id = make([]int, 0)
	for i := 0; i < len(committees_ref); i++ {
		nodes := utils.GetFieldValueforList(committees_ref[i].Nodes, "NodeID")
		for _, id := range nodes {
			allnodes_id = append(allnodes_id, id.(int))
		}
	}

	tbChain_ref.SetMessageHub(hub)
}
