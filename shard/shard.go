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
	chainDb    ethdb.Database // Block chain database
	leader     *Node
	txPool     *core.TxPool
	blockchain *core.BlockChain

	/* 计数器，初始等于客户端个数，每一个客户端发送注入完成信号时计数器减一 */
	injectNotDone int32
	connMap       map[string]net.Conn
	connMaplock   sync.RWMutex

	messageHub core.MessageHub
}

func NewShard(stack *Node, chainID int, clientCnt int) (*Shard, error) {
	// Node is used to maintain a blockchain with a database
	chainDb, err := stack.OpenDatabase("chaindata", 0, 0, "", false)
	if err != nil {
		return nil, err
	}

	genesis := core.DefaultGenesisBlock()
	genesisBlock := genesis.MustCommit(chainDb)
	if genesisBlock == nil {
		log.Error("NewShard genesisBlock MustCommit err")
	}

	chainConfig := &params.ChainConfig{
		ChainID: big.NewInt(int64(chainID)),
	}

	bc, err := core.NewBlockChain(chainDb, nil, chainConfig)
	if err != nil {
		log.Error("NewShard NewBlockChain err")
		return nil, err
	}

	pool := core.NewTxPool(bc)

	stack.chainID = chainID
	stack.addr = NodeTable[chainID]

	log.Info("NewShard", "chainID=", chainID, "addr=", stack.addr)

	shard := &Shard{
		chainDb:       chainDb,
		blockchain:    bc,
		txPool:        pool,
		leader:        stack,
		injectNotDone: int32(clientCnt),
		connMap:       make(map[string]net.Conn),
	}

	return shard, nil
}

func (shard *Shard) GetChainID() int {
	return shard.leader.chainID
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
		ShardID:    uint64(s.leader.chainID),
		BlockHash:  genesisBlock.Hash(),
		TxHash:     g_header.TxHash,
		StatusHash: g_header.Root,
	}
	s.messageHub.Send(core.MsgTypeAddTB, 0, tb, nil)
}
