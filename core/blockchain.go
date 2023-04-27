package core

import (
	"errors"
	"sync/atomic"
	"time"

	"go-w3chain/params"
	"go-w3chain/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	lru "github.com/hashicorp/golang-lru"
)

var (
	ErrNoGenesis = errors.New("genesis not found in chain")
	ErrUnknown   = errors.New("Unknown in blockchain")
)

const (
	blockCacheLimit       = 256
	TriesInMemoryInterval = 128
)

type txURI uint64

type BlockChain struct {
	chainConfig *params.ChainConfig // Chain configuration
	cacheConfig *CacheConfig        // Cache configuration for pruning

	db         ethdb.Database
	stateCache state.Database // State database to reuse between imports (contains state cache)

	statedb *state.StateDB // if 从简，直接用statedb

	genesisBlock *Block

	blockCache *lru.Cache // Cache for the most recent entire blocks

	addrInfo *utils.AddressInfo

	currentBlock atomic.Value // Current head of the block chain
}

// 设置 stateDB 的 config
// CacheConfig contains the configuration values for the trie caching/pruning
// that's resident in a blockchain.
type CacheConfig struct {
	TrieCleanLimit      int           // Memory allowance (MB) to use for caching trie nodes in memory
	TrieCleanJournal    string        // Disk journal for saving clean cache entries.
	TrieCleanRejournal  time.Duration // Time interval to dump clean cache to disk periodically
	TrieCleanNoPrefetch bool          // Whether to disable heuristic state prefetching for followup blocks
	TrieDirtyLimit      int           // Memory limit (MB) at which to start flushing dirty trie nodes to disk
	TrieDirtyDisabled   bool          // Whether to disable trie write caching and GC altogether (archive node)
	TrieTimeLimit       time.Duration // Time limit after which to flush the current in-memory trie to disk
	SnapshotLimit       int           // Memory allowance (MB) to use for caching snapshot entries in memory
	Preimages           bool          // Whether to store preimage of trie key to the disk

	SnapshotWait bool // Wait for snapshot construction on startup. TODO(karalabe): This is a dirty hack for testing, nuke it
}

// defaultCacheConfig are the default caching values if none are specified by the
// user (also used during testing).
var defaultCacheConfig = &CacheConfig{
	TrieCleanLimit: 256,
	TrieDirtyLimit: 256,
	TrieTimeLimit:  5 * time.Minute,
	SnapshotLimit:  256,
	SnapshotWait:   true,
}

func NewBlockChain(db ethdb.Database, cacheConfig *CacheConfig, chainConfig *params.ChainConfig, addrInfo *utils.AddressInfo) (*BlockChain, error) {
	if cacheConfig == nil {
		cacheConfig = defaultCacheConfig
	}

	blockCache, _ := lru.New(blockCacheLimit)

	bc := &BlockChain{
		chainConfig: chainConfig,
		cacheConfig: cacheConfig,
		db:          db,
		stateCache: state.NewDatabaseWithConfig(db, &trie.Config{
			Cache:     cacheConfig.TrieCleanLimit,
			Journal:   cacheConfig.TrieCleanJournal,
			Preimages: cacheConfig.Preimages,
		}),
		statedb:    NewStateDB(db),
		blockCache: blockCache,
		addrInfo:   addrInfo,
	}

	bc.genesisBlock = bc.GetBlockByNumber(0)
	if bc.genesisBlock == nil {
		return nil, ErrNoGenesis
	}

	// var nilBlock *Block
	bc.currentBlock.Store(bc.genesisBlock)

	head := bc.CurrentBlock()
	if _, err := state.New(head.Root(), bc.stateCache, nil); err != nil {
		return nil, ErrUnknown
	}

	return bc, nil
}

func (bc *BlockChain) GetChainHeight() uint64 {
	return bc.CurrentBlock().NumberU64()
}

func (bc *BlockChain) GetAddrTable() *map[common.Address]int {
	// return bc.addressTable
	return &bc.addrInfo.AddrTable
}

func (bc *BlockChain) GetBlockChainAddrInfo() *utils.AddressInfo {
	return bc.addrInfo
}
func (bc *BlockChain) GetStateDB() *state.StateDB {
	return bc.statedb
}

func (bc *BlockChain) WriteBlock(block *Block) {
	bc.currentBlock.Store(block)
}
