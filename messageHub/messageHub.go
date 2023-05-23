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
	"math/rand"
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
	case core.MsgTypeShardReply2Client:
		client := clients_ref[id]
		receipts := msg.([]*result.TXReceipt)
		client.AddTXReceipts(receipts)

	case core.MsgTypeClientInjectTX2Committee:
		com := committees_ref[id]
		txs := msg.([]*core.Transaction)
		com.InjectTXs(txs)

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
	case core.MsgTypeComGetState:
		shard := shards_ref[id]
		states := shard.GetBlockChain().GetStateDB()
		parentHeight := shard.GetBlockChain().CurrentBlock().Number()
		callback(states, parentHeight)
	case core.MsgTypeAddBlock2Shard:
		shard := shards_ref[id]
		block := msg.(*core.Block)
		shard.GetBlockChain().WriteBlock(block)
	case core.MsgTypeReady4Reconfig:
		toReconfig()
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

func toReconfig() {
	isNotToReconfig -= 1
	if isNotToReconfig == 0 {
		// 这里需要开一个新线程，不要与最后一个调用此函数的委员会共用一个线程
		go reconfig()
		isNotToReconfig = len(committees_ref)
	}
}

/* 打乱allnodes_id，重新设定委员会中的节点 */
func reconfig() {
	// 设置随机种子
	rand.Seed(int64(getSeed()))

	rand.Shuffle(len(allnodes_id), func(i, j int) {
		allnodes_id[i], allnodes_id[j] = allnodes_id[j], allnodes_id[i]
	})

	begin := 0
	for _, com := range committees_ref {
		node_num := len(com.Nodes)
		nodes_index := allnodes_id[begin : begin+node_num]
		begin += node_num

		nodes_4_com := make([]*core.Node, node_num)
		for j, index := range nodes_index {
			nodes_4_com[j] = nodes_ref[index]
		}

		com.SetReconfigRes(nodes_4_com)
	}
}

/* 从信标链获取最新高度的已确认区块（比如有足够数量的区块跟在后面）的信息，返回一个整数作为重组的种子 */
func getSeed() int {
	seed := rand.Int()
	// TODO: implement me
	return seed
}
