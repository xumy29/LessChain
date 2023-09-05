package shard

import (
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/node"
	"go-w3chain/params"
	"math/big"
	"net"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

type Shard struct {
	shardID uint32
	chainDB ethdb.Database // Block chain database
	leader  *node.Node
	nodes   []*node.Node
	// txPool     *core.TxPool
	blockchain *core.BlockChain

	connMap     map[string]net.Conn
	connMaplock sync.RWMutex

	txStatus map[uint64]uint64

	messageHub core.MessageHub
}

func NewShard(shardID uint32, _node *node.Node) *Shard {
	// 获取节点的数据库
	chainDB := _node.GetDB()

	genesisBlock := core.DefaultGenesisBlock()
	genesisBlock.MustCommit(chainDB)

	chainConfig := &params.ChainConfig{
		ChainID: big.NewInt(int64(shardID)),
	}

	bc, err := core.NewBlockChain(chainDB, nil, chainConfig)
	if err != nil {
		log.Error("NewShard NewBlockChain err", "err", err)
	}

	log.Info("NewShard", "shardID", shardID,
		"nodeID", _node.NodeID)

	shard := &Shard{
		nodes:      []*node.Node{_node},
		shardID:    shardID,
		chainDB:    chainDB,
		blockchain: bc,
		connMap:    make(map[string]net.Conn),
		txStatus:   make(map[uint64]uint64),
	}

	if isLeader(_node) {
		shard.setLeader(_node)
	}

	return shard
}

func isLeader(node *node.Node) bool {
	return node.NodeID == 0
}

func (shard *Shard) setLeader(node *node.Node) {
	shard.leader = node
}

func (shard *Shard) GetShardID() uint32 {
	return shard.shardID
}

func (shard *Shard) AddBlock(block *core.Block) {
	shard.blockchain.WriteBlock(block)
	for _, tx := range block.Body().Transactions {
		shard.txStatus[tx.ID] = tx.TXStatus
	}
}

func (shard *Shard) GetBlockChain() *core.BlockChain {
	return shard.blockchain
}

func (shard *Shard) GetChainHeight() uint64 {
	return shard.blockchain.GetChainHeight()
}

func (shard *Shard) SetInitialAccountState(Addrs map[common.Address]struct{}, maxValue *big.Int) {
	statedb := shard.blockchain.GetStateDB()
	for addr := range Addrs {
		statedb.AddBalance(addr, maxValue)
		if curValue := statedb.GetBalance(addr); curValue.Cmp(maxValue) != 0 {
			log.Error("Opps, something wrong!", "curValue", curValue, "Set maxValue", maxValue)
		}

	}
}

func (s *Shard) SetMessageHub(hub core.MessageHub) {
	s.messageHub = hub
}

func (s *Shard) Start() {
	s.addGenesisTB()
}

func (s *Shard) Close() {
	// todo: fill it
}

func (s *Shard) HandleComGetState(request *core.ComGetState) {
	response := &core.ShardSendState{
		StateDB: s.blockchain.GetStateDB(),
		Height:  s.blockchain.CurrentBlock().Number(),
	}

	s.messageHub.Send(core.MsgTypeShardSendStateToCom, request.From_comID, response, nil)
}

/**
 * 将创世区块的信标写到信标链
 * 该方法在分片被创建后，委员会启动前被调用
 */
func (s *Shard) addGenesisTB() {
	/* 写入到信标链 */
	genesisBlock := s.blockchain.CurrentBlock()
	g_header := genesisBlock.Header()
	tb := &core.TimeBeacon{
		Height:     g_header.Number.Uint64(),
		ShardID:    uint32(s.shardID),
		BlockHash:  genesisBlock.Hash().Hex(),
		TxHash:     g_header.TxHash.Hex(),
		StatusHash: g_header.Root.Hex(),
	}
	signedTb := &core.SignedTB{
		TimeBeacon: *tb,
		// 创世区块暂时不需要签名
	}
	addrs := make([]common.Address, 0)
	for _, node := range s.nodes {
		addrs = append(addrs, *node.GetAccount().GetAccountAddress())
	}
	s.messageHub.Send(core.MsgTypeCommitteeInitialAddrs, s.shardID, addrs, nil)

	s.messageHub.Send(core.MsgTypeCommitteeAddTB, 0, signedTb, nil)
}
