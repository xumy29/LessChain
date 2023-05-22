package messageHub

import (
	"go-w3chain/beaconChain"
	"go-w3chain/client"
	"go-w3chain/committee"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"go-w3chain/shard"
	"math/rand"
)

type GoodMessageHub struct {
	mid int
}

/* 是controller中的shards和clients的引用 */
var shards_ref []*shard.Shard
var committees_ref []*committee.Committee
var clients_ref []*client.Client
var tbChain_ref *beaconChain.BeaconChain

/* 初始等于委员会数量，一个委员会调用ToReconfig时该变量减一，减到0时触发重组 */
var isNotToReconfig int

/* 委员会ID到分片ID的映射 */
var comMap []int

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
	case core.MsgTypeComGetTX:
		shard := shards_ref[id]
		blockCap := msg.(int)
		txs := shard.TXpool().Pending(blockCap)
		states := shard.GetBlockChain().GetStateDB()
		parentHeight := shard.GetBlockChain().CurrentBlock().Number()
		callback(txs, states, parentHeight)
	case core.MsgTypeAddBlock2Shard:
		shard := shards_ref[id]
		block := msg.(*core.Block)
		shard.GetBlockChain().WriteBlock(block)
	case core.MsgTypeReady4Reconfig:
		toReconfig()
	}
}

func (hub *GoodMessageHub) Init(clients []*client.Client, shards []*shard.Shard, committees []*committee.Committee, tbChain *beaconChain.BeaconChain) {
	clients_ref = clients
	shards_ref = shards
	committees_ref = committees
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
	// 根据committee的初始映射情况初始化comMap，之后重组时打乱此数组
	comMap = make([]int, len(committees_ref))
	for i := range comMap {
		comMap[i] = int(committees_ref[i].GetShardID())
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

/* 打乱conMap，重新设定委员会到分片的映射 */
func reconfig() {
	// 设置随机种子
	rand.Seed(int64(getSeed()))

	rand.Shuffle(len(comMap), func(i, j int) {
		comMap[i], comMap[j] = comMap[j], comMap[i]
	})
	for i, com := range committees_ref {
		com.SetReconfigRes(comMap[i])
	}
}

/* 从信标链获取最新高度的已确认区块（比如有足够数量的区块跟在后面）的信息，返回一个整数作为重组的种子 */
func getSeed() int {
	seed := rand.Int()
	// TODO: implement me
	return seed
}
