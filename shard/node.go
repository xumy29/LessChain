package shard

import (
	"go-w3chain/log"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

type Node struct {
	config *NodeConfig
	keyDir string // key store directory

	lock      sync.Mutex
	databases map[*closeTrackingDB]struct{} // All open databases, 只关心 key，不关注值。

	addr    string
	chainID int
}

func New(conf *NodeConfig) *Node {
	confcopy := *conf
	conf = &confcopy

	node := &Node{
		config:    conf,
		databases: make(map[*closeTrackingDB]struct{}),
	}

	node.keyDir = getKeyStoreDir(conf)

	return node

}

// OpenDatabase opens an existing database with the given name (or creates one if no
// previous can be found) from within the node's instance directory.
func (n *Node) OpenDatabase(name string, cache, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	n.lock.Lock()
	defer n.lock.Unlock()

	// namepsace = "", file = /home/pengxiaowen/.brokerChain/xxx/name
	// cache , handle = 0, readonly = false
	var db ethdb.Database
	var err error
	db, err = rawdb.NewLevelDBDatabase(n.ResolvePath(name), cache, handles, namespace, readonly)

	log.Trace("Database", "node dataDir", n.config.DataDir)
	// log.Trace("Database", "node keyDir", n.keyDir)
	// log.Trace("Database", "node chaindata", n.ResolvePath(name))

	return db, err

}

// ResolvePath returns the absolute path of a resource in the instance directory.
func (n *Node) ResolvePath(x string) string {
	return filepath.Join(n.config.DataDir, x)
}

// closeTrackingDB wraps the Close method of a database. When the database is closed by the
// service, the wrapper removes it from the node's database map. This ensures that Node
// won't auto-close the database if it is closed by the service that opened it.
type closeTrackingDB struct {
	ethdb.Database
	n *Node
}

func (db *closeTrackingDB) Close() error {
	db.n.lock.Lock()
	delete(db.n.databases, db)
	db.n.lock.Unlock()
	return db.Database.Close()
}

// wrapDatabase ensures the database will be auto-closed when Node is closed.
func (n *Node) wrapDatabase(db ethdb.Database) ethdb.Database {
	wrapper := &closeTrackingDB{db, n}
	n.databases[wrapper] = struct{}{}
	return wrapper
}

// closeDatabases closes all open databases.
func (n *Node) closeDatabases() (errors []error) {
	for db := range n.databases {
		delete(n.databases, db)
		if err := db.Database.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}
