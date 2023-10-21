package node

import (
	"bytes"
	"fmt"
	"go-w3chain/cfg"
	"go-w3chain/core"
	"go-w3chain/eth_chain"
	"go-w3chain/log"
	"go-w3chain/pbft"
	"go-w3chain/utils"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

type Node struct {
	NodeInfo         *core.NodeInfo
	nodeSendInfoLock sync.Mutex

	/** 该节点对应的账户 */
	w3Account *W3Account

	/** 存储该节点所有数据的目录，包括chaindata和keystore
	 * $Home/.w3Chain/shardi/nodej/
	 */
	DataDir string

	db ethdb.Database

	shard         core.Shard
	shardNum      int
	comAllNodeNum int // 委员会中所有节点，包括共识节点和非共识节点
	com           core.Committee

	/* 节点上一次运行vrf得到的结果 */
	VrfValue []byte
	pbftNode *pbft.PbftConsensusNode

	contractAddr common.Address
	contractAbi  *abi.ABI

	messageHub core.MessageHub

	reconfigResLock     sync.Mutex
	reconfigResult      *core.ReconfigResult                // 本节点的重组结果
	reconfigResults     []*core.ReconfigResult              // 本委员会所有节点的重组结果
	com2ReconfigResults map[uint32]*core.ComReconfigResults // 所有委员会的节点的重组结果
}

func NewNode(parentdataDir string, shardNum, shardID, comID, nodeID, shardSize, comAllNodeNum int) *Node {
	nodeInfo := &core.NodeInfo{
		ShardID:  uint32(shardID),
		ComID:    uint32(comID),
		NodeID:   uint32(nodeID),
		NodeAddr: cfg.ComNodeTable[uint32(shardID)][uint32(nodeID)],
	}
	node := &Node{
		DataDir:       filepath.Join(parentdataDir, fmt.Sprintf("S%dN%d", shardID, nodeID)),
		shardNum:      shardNum,
		NodeInfo:      nodeInfo,
		comAllNodeNum: comAllNodeNum,
	}

	node.w3Account = NewW3Account(node.DataDir)
	printAccounts(node.w3Account)

	db, err := node.OpenDatabase("chaindata", 0, 0, "", false)
	if err != nil {
		log.Error("open database fail", "nodeID", nodeID)
	}
	node.db = db

	// 节点刚创建时，shardID == ComID
	node.pbftNode = pbft.NewPbftNode(node.NodeInfo, uint32(shardSize), "")

	return node
}

func (node *Node) SetMessageHub(hub core.MessageHub) {
	node.messageHub = hub
}

func (node *Node) Start() {
	node.com.Start(node.NodeInfo.NodeID)
	node.sendNodeInfo()
}

func (node *Node) sendNodeInfo() {
	if utils.IsComLeader(node.NodeInfo.NodeID) {
		return
	}
	info := &core.NodeSendInfo{
		NodeInfo: node.NodeInfo,
		Addr:     node.w3Account.accountAddr,
	}
	log.Debug(fmt.Sprintf("sendNodeInfo... addr: %x", info.Addr))
	node.messageHub.Send(core.MsgTypeNodeSendInfo2Leader, node.NodeInfo.ComID, info, nil)
}

func (node *Node) RunPbft(block *core.Block, exit chan struct{}) {
	node.pbftNode.SetSequenceID(block.NumberU64())
	node.pbftNode.Propose(block)
	// wait till consensus is complete

	select {
	case <-node.pbftNode.OneConsensusDone:
		return
	case <-exit:
		exit <- struct{}{}
		return
	}

}

func (node *Node) SetShard(shard core.Shard) {
	node.shard = shard
}

func (node *Node) SetCommittee(com core.Committee) {
	node.com = com
}

func (node *Node) GetShard() core.Shard {
	return node.shard
}

func (node *Node) GetCommittee() core.Committee {
	return node.com
}

func (node *Node) GetPbftNode() *pbft.PbftConsensusNode {
	return node.pbftNode
}

func (node *Node) Close() {
	node.CloseDatabase()
}

// ResolvePath returns the absolute path of a resource in the instance directory.
func (n *Node) ResolvePath(x string) string {
	return filepath.Join(n.DataDir, x)
}

func (n *Node) GetDB() ethdb.Database {
	return n.db
}

func (n *Node) GetAddr() string {
	return n.NodeInfo.NodeAddr
}

func (n *Node) GetAccount() *W3Account {
	return n.w3Account
}

func (n *Node) HandleNodeSendInfo(info *core.NodeSendInfo) {
	n.nodeSendInfoLock.Lock()
	defer n.nodeSendInfoLock.Unlock()

	n.shard.AddInitialAddr(info.Addr, info.NodeInfo.NodeID)
	if len(n.shard.GetNodeAddrs()) == int(n.comAllNodeNum) {
		n.shard.Start()
	}
}

func (n *Node) HandleBooterSendContract(data *core.BooterSendContract) {
	n.contractAddr = data.Addr
	contractABI, err := abi.JSON(strings.NewReader(eth_chain.MyContractABI()))
	if err != nil {
		log.Error("get contracy abi fail", "err", err)
	}
	n.contractAbi = &contractABI
	// 启动 worker，满足三个条件： 1.是leader节点；2.收到合约地址；3.和委员会内所有节点建立起联系
	if utils.IsComLeader(n.NodeInfo.NodeID) {
		n.com.StartWorker()
	}
}

type HandleReconfigMsgs struct {
}

// leader节点调用
func (n *Node) AddReconfigResult(res *core.ReconfigResult) {
	if n.NodeInfo.NodeID != 0 {
		log.Error(fmt.Sprintf("wrong invoking function. only leader node can invoke this function. curNodeInfo: %v", n.NodeInfo))
	}

	if n.reconfigResults == nil { // 第一次重组时
		n.reconfigResults = make([]*core.ReconfigResult, 0)
		n.reconfigResults = append(n.reconfigResults, res)
	} else {
		last := n.reconfigResults[len(n.reconfigResults)-1]
		if last.SeedHeight < res.SeedHeight { // 是上次重组时留下的
			n.reconfigResults = make([]*core.ReconfigResult, 0)
			n.reconfigResults = append(n.reconfigResults, res)
		} else { // 是当前重组
			n.reconfigResults = append(n.reconfigResults, res)
		}
	}
}

// leader节点调用
func (n *Node) AddReconfigResults(res *core.ComReconfigResults) {
	if n.com2ReconfigResults == nil { // 第一次重组时
		n.com2ReconfigResults = make(map[uint32]*core.ComReconfigResults)
		n.com2ReconfigResults[res.ComID] = res
	} else {
		for _, last := range n.com2ReconfigResults {
			if last.Results[0].SeedHeight < res.Results[0].SeedHeight { // 是上次重组时留下的
				n.com2ReconfigResults = make(map[uint32]*core.ComReconfigResults)
				n.com2ReconfigResults[res.ComID] = res
			} else { // 是当前重组
				if _, ok := n.com2ReconfigResults[res.ComID]; ok {
					log.Error(fmt.Sprintf("this committee's reconfig result has been received. why receive again? from_comID: %d", res.ComID))
				}
				n.com2ReconfigResults[res.ComID] = res
			}
			break // 拿到任意一个元素后即可结束遍历
		}
	}
}

func (n *Node) InitReconfig(data *core.InitReconfig) {
	n.com.SetOldTxPool()
	data.ComNodeNum = uint32(n.comAllNodeNum)
	n.messageHub.Send(core.MsgTypeLeaderInitReconfig, n.NodeInfo.ComID, data, nil)
}

func (n *Node) HandleLeaderInitReconfig(data *core.InitReconfig) {
	n.com.UpdateTbChainHeight(data.SeedHeight)

	acc := n.GetAccount()
	vrfValue := acc.GenerateVRFOutput(data.Seed[:]).RandomValue
	newComId := utils.VrfValue2Shard(vrfValue, uint32(n.shardNum))

	reply := &core.ReconfigResult{
		Seed:         data.Seed,
		SeedHeight:   data.SeedHeight,
		Vrf:          vrfValue,
		Addr:         acc.accountAddr,
		OldNodeInfo:  n.NodeInfo,
		Belong_ComID: data.ComID,
		NewComID:     newComId,
	}
	n.reconfigResult = reply
	n.messageHub.Send(core.MsgTypeSendReconfigResult2ComLeader, data.ComID, reply, nil)
}

func (n *Node) HandleSendReconfigResult2ComLeader(data *core.ReconfigResult) {
	// 省略对vrf的检查...

	n.reconfigResLock.Lock()

	n.AddReconfigResult(data)
	if len(n.reconfigResults) == int(n.comAllNodeNum) {
		res := &core.ComReconfigResults{
			ComID:      n.NodeInfo.ComID,
			Results:    n.reconfigResults,
			ComNodeNum: uint32(n.comAllNodeNum),
		}
		// 发给自己也用网络，不直接存，这样可以统一处理
		n.reconfigResLock.Unlock()
		n.messageHub.Send(core.MsgTypeSendReconfigResults2AllComLeaders, n.NodeInfo.ComID, res, nil)
	} else {
		n.reconfigResLock.Unlock()
	}
}

func (n *Node) HandleSendReconfigResults2AllComLeaders(data *core.ComReconfigResults) {
	// 省略对vrf的检查...

	n.reconfigResLock.Lock()
	defer n.reconfigResLock.Unlock()

	n.AddReconfigResults(data)
	if len(n.com2ReconfigResults) == n.shardNum {
		// 将所有vrf结果发送给委员会内的节点，包括发送者leader本身
		n.messageHub.Send(core.MsgTypeSendReconfigResults2ComNodes, n.NodeInfo.ComID, n.com2ReconfigResults, nil)
	}

}

func (n *Node) HandleSendReconfigResults2ComNodes(data *map[uint32]*core.ComReconfigResults) {
	// 省略对vrf的检查...

	// 先得到每个新委员会中的节点结果
	newCom2Results := make(map[uint32][]*core.ReconfigResult)
	for _, rets := range *data {
		for _, res := range rets.Results {
			newCom2Results[res.NewComID] = append(newCom2Results[res.NewComID], res)
		}
	}
	// 对每个新委员会中结果按vrf从小到大排序
	for _, results := range newCom2Results {
		sort.Slice(results, func(i, j int) bool {
			return bytes.Compare(results[i].Vrf, results[j].Vrf) < 0
		})
	}

	// 更新ComNodeTable和本节点的NodeInfo
	localNodeInfo := n.NodeInfo
	newComNodeTable := make(map[uint32]map[uint32]string)
	var oldComLeaderAddr string
	var i uint32
	for i = 0; i < uint32(n.shardNum); i++ {
		newComNodeTable[i] = make(map[uint32]string)
		for newID, result := range newCom2Results[i] {
			newComNodeTable[i][uint32(newID)] = result.OldNodeInfo.NodeAddr
			if *result.OldNodeInfo == *localNodeInfo {
				newNodeInfo := &core.NodeInfo{
					ShardID:  localNodeInfo.ShardID,
					ComID:    n.reconfigResult.NewComID,
					NodeID:   uint32(newID),
					NodeAddr: localNodeInfo.NodeAddr,
				}
				log.Debug(fmt.Sprintf("local nodeInfo updated... before reconfiguration: %v after: %v", localNodeInfo, newNodeInfo))
				n.updateNodeInfo(newNodeInfo)
			}
			if result.OldNodeInfo.ComID == n.reconfigResult.NewComID && result.OldNodeInfo.NodeID == 0 { // 本节点所在的新委员会原来的leader
				oldComLeaderAddr = result.OldNodeInfo.NodeAddr
			}
		}
	}

	cfg.ComNodeTable = newComNodeTable
	// 各分片的新leader
	com2Leader := make(map[uint32]string)
	for shardID, list := range cfg.ComNodeTable {
		com2Leader[shardID] = list[0]
	}
	log.Debug(fmt.Sprintf("NewComLeaders：%v", com2Leader))
	for shardID, list := range cfg.ComNodeTable {
		log.Debug(fmt.Sprintf("comID: %d nodeAddrs: %v", shardID, list))
	}

	n.EndReconfig(newCom2Results, oldComLeaderAddr)
}

func (n *Node) updateNodeInfo(newNodeInfo *core.NodeInfo) {
	n.NodeInfo = newNodeInfo
	n.pbftNode.NodeInfo = newNodeInfo
}

func (n *Node) EndReconfig(newCom2Results map[uint32][]*core.ReconfigResult, oldComLeaderAddr string) {
	// 更新委员会节点数量
	n.comAllNodeNum = len(newCom2Results[n.NodeInfo.ComID])
	log.Debug(fmt.Sprintf("after reconfiguration, com %d has %d nodes in total.", n.NodeInfo.ComID, n.comAllNodeNum))
	if n.comAllNodeNum < 4 { // pbft协议至少需要包括leader在内的3个节点
		report := &core.ErrReport{
			NodeAddr: n.NodeInfo.NodeAddr,
			Err:      fmt.Sprintf("after reconfiguration, com %d has less than 4 nodes, not enough for pbft consensus", n.NodeInfo.ComID),
		}
		n.messageHub.Send(core.MsgTypeReportError, 0, report, nil)
		log.Error(fmt.Sprintf("after reconfiguration, com %d has less than 4 nodes, not enough for pbft consensus", n.NodeInfo.ComID))
	}

	// 更新合约上的地址
	if utils.IsComLeader(n.NodeInfo.NodeID) {
		comResults := newCom2Results[n.NodeInfo.ComID]
		addrs := make([]common.Address, 0)
		vrfs := make([][]byte, 0)
		for _, res := range comResults {
			addrs = append(addrs, res.Addr)
			vrfs = append(vrfs, res.Vrf)
		}
		n.com.AdjustRecordedAddrs(addrs, vrfs, comResults[0].SeedHeight)
	}

	// 重组开始时已经调用过一次，此处再次调用，是因为重组过程节点可能继续收到客户端发送的交易
	n.com.SetOldTxPool()

	// 重新启动委员会和worker、新建交易池
	n.com.Start(n.NodeInfo.NodeID)

	if utils.IsComLeader(n.NodeInfo.NodeID) { // 每个委员会的leader都会给客户端发送新表
		n.messageHub.Send(core.MsgTypeSendNewNodeTable2Client, 0, cfg.ComNodeTable, nil)
	}
	// 同步交易池
	if utils.IsComLeader(n.NodeInfo.NodeID) {
		if n.NodeInfo.NodeAddr == oldComLeaderAddr {
			poolTx := n.com.HandleGetPoolTx(nil)
			n.com.SetPoolTx(poolTx)
		} else {
			getPoolTxsCh := make(chan struct{}, 1)
			callback := func(res ...interface{}) {
				poolTx := res[0].(*core.PoolTx)
				n.com.SetPoolTx(poolTx)
				getPoolTxsCh <- struct{}{}
			}
			request := &core.GetPoolTx{
				ServerAddr:   oldComLeaderAddr,
				ClientAddr:   n.NodeInfo.NodeAddr,
				RequestComID: n.NodeInfo.ComID,
			}

			n.messageHub.Send(core.MsgTypeGetPoolTx, n.NodeInfo.ComID, request, callback)
			// 等待交易池更新后再启动worker
			<-getPoolTxsCh
		}
	}

	// 重置pbft
	if utils.IsComLeader(n.NodeInfo.NodeID) {
		n.pbftNode.Reset()
	}

	if utils.IsComLeader(n.NodeInfo.NodeID) {
		n.com.StartWorker()
	}

	// 删除无用的长连接，释放系统资源，防止某些端口被强制关闭
	// n.messageHub.Send(core.MsgTypeClearConnection, 0, n.NodeInfo, nil)

	// todo
}

/*
	//////////////////////////////////////////////////////////////
	节点的数据库相关的操作，包括打开、关闭等
	/////////////////////////////////////////////////////////////
*/

// OpenDatabase opens an existing database with the given name (or creates one if no
// previous can be found) from within the node's instance directory.
func (n *Node) OpenDatabase(name string, cache, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	// namepsace = "", file = /home/pengxiaowen/.brokerChain/xxx/name
	// cache , handle = 0, readonly = false
	var err error
	n.db, err = rawdb.NewLevelDBDatabase(n.ResolvePath(name), cache, handles, namespace, readonly)

	log.Trace("openDatabase", "node dataDir", n.DataDir)
	// log.Trace("Database", "node keyDir", n.keyDir)
	// log.Trace("Database", "node chaindata", n.ResolvePath(name))

	return n.db, err
}

func (n *Node) CloseDatabase() {
	err := n.db.Close()
	if err != nil {
		log.Error("closeDatabase fail.", "nodeInfo", n.NodeInfo)
	}
	// log.Debug("closeDatabase", "nodeID", n.NodeInfo.NodeID)
}
