package shard

import (
	"go-w3chain/beaconChain"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/params"
	"math/big"
	"net"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

type Shard struct {
	shardID    int
	chainDB    ethdb.Database // Block chain database
	leader     *core.Node
	nodes      []*core.Node
	txPool     *core.TxPool
	blockchain *core.BlockChain

	/* 计数器，初始等于客户端个数，每一个客户端发送注入完成信号时计数器减一 */
	injectNotDone int32
	connMap       map[string]net.Conn
	connMaplock   sync.RWMutex

	messageHub core.MessageHub
}

func NewShard(nodes []*core.Node, shardID int, clientCnt int) (*Shard, error) {
	if len(nodes) == 0 {
		log.Error("number of nodes should be larger than 0.")
	}
	// 将分片中第一个节点作为leader节点，取其数据库作为分片的数据库
	// 其他节点的数据库默认不会被更新
	chainDB := nodes[0].GetDB()

	genesis := core.DefaultGenesisBlock()
	genesisBlock := genesis.MustCommit(chainDB)
	if genesisBlock == nil {
		log.Error("NewShard genesisBlock MustCommit err")
	}

	chainConfig := &params.ChainConfig{
		ChainID: big.NewInt(int64(shardID)),
	}

	bc, err := core.NewBlockChain(chainDB, nil, chainConfig)
	if err != nil {
		log.Error("NewShard NewBlockChain err")
		return nil, err
	}

	pool := core.NewTxPool(bc)

	log.Info("NewShard", "shardID", shardID, "nodes", nodes, "leader", nodes[0])

	shard := &Shard{
		nodes:         nodes,
		shardID:       shardID,
		chainDB:       chainDB,
		blockchain:    bc,
		txPool:        pool,
		leader:        nodes[0],
		injectNotDone: int32(clientCnt),
		connMap:       make(map[string]net.Conn),
	}

	return shard, nil
}

func (shard *Shard) GetChainID() int {
	return shard.shardID
}

func (shard *Shard) GetBlockChain() *core.BlockChain {
	return shard.blockchain
}

func (shard *Shard) GetChainHeight() uint64 {
	return shard.blockchain.GetChainHeight()
}

func (shard *Shard) SetInitialState(Addrs map[common.Address]struct{}, maxValue *big.Int) {
	statedb := shard.blockchain.GetStateDB()
	for addr := range Addrs {
		statedb.AddBalance(addr, maxValue)
		if curValue := statedb.GetBalance(addr); curValue.Cmp(maxValue) != 0 {
			log.Error("Opps, something wrong!", "curValue", curValue, "Set maxValue", maxValue)
		}

	}
}

func (shard *Shard) InjectTXs(txs []*core.Transaction) {
	shard.txPool.AddTxs(txs)
}

func (shard *Shard) SetInjectTXDone() {
	atomic.AddInt32(&shard.injectNotDone, -1)
}

/* 交易注入完成即可停止 */
func (shard *Shard) CanStopV2() bool {
	return shard.injectNotDone == 0
}

func (s *Shard) TXpool() *core.TxPool {
	return s.txPool
}

func (s *Shard) SetMessageHub(hub core.MessageHub) {
	s.messageHub = hub
}

/**
 * 将创世区块的信标写到信标链
 * 该方法在分片被创建后，委员会启动前被调用
 */
func (s *Shard) AddGenesisTB() {
	/* 写入到信标链 */
	genesisBlock := s.blockchain.CurrentBlock()
	g_header := genesisBlock.Header()
	tb := &beaconChain.TimeBeacon{
		Height:     g_header.Number.Uint64(),
		ShardID:    uint64(s.shardID),
		BlockHash:  genesisBlock.Hash(),
		TxHash:     g_header.TxHash,
		StatusHash: g_header.Root,
	}
	s.messageHub.Send(core.MsgTypeAddTB, 0, tb, nil)
}
