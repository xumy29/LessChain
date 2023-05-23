package core

import (
	"fmt"
	"go-w3chain/log"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

const (
	datadir                = ".w3chain"
	datadirDefaultKeyStore = "keystore" // Path within the datadir to the keystore
)

func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := os.Getenv("HOME")
	return filepath.Join(home, datadir)

}

func DefaultKeyStoreDir(prefix string) string {
	keydir := filepath.Join(prefix, datadirDefaultKeyStore)
	return keydir
}

type NodeConfig struct {
	Name string

	WSHost string
	WSPort int
}

type Node struct {
	NodeID  int
	config  *NodeConfig
	keyDir  string // key store directory
	DataDir string

	lock sync.Mutex
	db   ethdb.Database

	shardID int
	/* 节点所在委员会的ID */
	commID int
}

func NewNode(conf *NodeConfig, dataDir string, shardID int, nodeID int) *Node {
	node := &Node{
		config:  conf,
		DataDir: dataDir,
		shardID: shardID,
		NodeID:  nodeID,
	}

	node.keyDir = DefaultKeyStoreDir(dataDir)
	db, err := node.OpenDatabase("chaindata", 0, 0, "", false)
	if err != nil {
		log.Error("open database fail", "nodeID", nodeID)
	}
	node.db = db

	return node

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
	return fmt.Sprintf("%s:%d", n.config.WSHost, n.config.WSPort)
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

	log.Debug("openDatabase", "node dataDir", n.DataDir)
	// log.Trace("Database", "node keyDir", n.keyDir)
	// log.Trace("Database", "node chaindata", n.ResolvePath(name))

	return n.db, err
}

func (n *Node) CloseDatabase() {
	err := n.db.Close()
	if err != nil {
		log.Error("closeDatabase fail.", "nodeConfig", n.config)
	}
	log.Debug("closeDatabase", "nodeID", n.NodeID)
}
