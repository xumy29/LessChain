package node

import (
	"fmt"
	"go-w3chain/cfg"
	"go-w3chain/core"
	"go-w3chain/eth_chain"
	"go-w3chain/log"
	"go-w3chain/pbft"
	"go-w3chain/utils"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

type Node struct {
	NodeID     int
	addrConfig *core.NodeAddrConfig

	// /** 存储该节点钱包数据的目录
	//  * $Home/.w3Chain/shardi/nodej/keystore
	//  */
	// keyDir string

	/** 该节点对应的账户 */
	w3Account *W3Account

	/** 存储该节点所有数据的目录，包括chaindata和keystore
	 * $Home/.w3Chain/shardi/nodej/
	 */
	DataDir string

	lock sync.Mutex
	db   ethdb.Database

	shardID int
	shard   core.Shard
	comID   int
	com     core.Committee
	/* 节点所在委员会的ID */
	commID int
	/* 节点上一次运行vrf得到的结果 */
	VrfValue []byte
	pbftNode *pbft.PbftConsensusNode

	contractAddr common.Address
	contractAbi  *abi.ABI

	messageHub core.MessageHub
}

func NewNode(conf *core.NodeAddrConfig, parentdataDir string, shardID, nodeID, shardSize int) *Node {
	node := &Node{
		addrConfig: conf,
		DataDir:    filepath.Join(parentdataDir, conf.Name),
		shardID:    shardID,
		commID:     shardID,
		NodeID:     nodeID,
	}

	node.w3Account = NewW3Account(node.DataDir)
	printAccounts(node.w3Account)

	db, err := node.OpenDatabase("chaindata", 0, 0, "", false)
	if err != nil {
		log.Error("open database fail", "nodeID", nodeID)
	}
	node.db = db

	// 节点刚创建时，shardID == ComID
	node.pbftNode = pbft.NewPbftNode(uint32(shardID), uint32(shardID), uint32(nodeID), uint32(shardSize),
		cfg.AllProtocolChanges, "")

	return node
}

func (node *Node) SetMessageHub(hub core.MessageHub) {
	node.messageHub = hub
}

func (node *Node) Start() {
	node.com.Start(node.NodeID)
	node.sendNodeInfo()
}

func (node *Node) sendNodeInfo() {
	if utils.IsComLeader(node.NodeID) {
		return
	}
	info := &core.NodeSendInfo{
		NodeInfo: node.pbftNode.NodeInfo,
		Addr:     node.w3Account.accountAddr,
	}
	node.messageHub.Send(core.MsgTypeNodeSendInfo2Leader, uint32(node.comID), info, nil)
}

func (node *Node) PbftPropose(block *core.Block) {
	node.pbftNode.Propose(block)
	// wait till consensus is complete
	<-node.pbftNode.OneConsensusDone
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
	return fmt.Sprintf("%s:%d", n.addrConfig.Host, n.addrConfig.Port)
}

func (n *Node) GetAccount() *W3Account {
	return n.w3Account
}

func (n *Node) HandleNodeSendInfo(info *core.NodeSendInfo) {
	n.com.AddMember(info.NodeInfo)
	if len(n.com.GetMembers()) == int(n.pbftNode.GetNodes_num()) {
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
	if utils.IsComLeader(n.NodeID) {
		if len(n.com.GetMembers()) != int(n.pbftNode.GetNodes_num()) {
			log.Error(fmt.Sprintf("incorrect nodes number in committee. should be %v, got %v", n.pbftNode.GetNodes_num(), len(n.com.GetMembers())))
		}
		n.com.StartWorker()
	}
}

/*
	//////////////////////////////////////////////////////////////
	节点的数据库相关的操作，包括打开、关闭等
	/////////////////////////////////////////////////////////////
*/

// OpenDatabase opens an existing database with the given name (or creates one if no
// previous can be found) from within the node's instance directory.
func (n *Node) OpenDatabase(name string, cache, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	n.lock.Lock()
	defer n.lock.Unlock()

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
		log.Error("closeDatabase fail.", "nodeConfig", n.addrConfig)
	}
	// log.Debug("closeDatabase", "nodeID", n.NodeID)
}
