package shard

import (
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/miner"
	"go-w3chain/params"
	"go-w3chain/utils"
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
	miner      *miner.Miner
	headerCh   <-chan struct{} // shard接收来自 headerCh 的消息
	exitCh     chan struct{}
	/* 计数器，初始等于客户端个数，每一个客户端发送注入完成信号时计数器减一 */
	injectNotDone int32
	connMap       map[string]net.Conn
	connMaplock   sync.RWMutex

	wg sync.WaitGroup
}

func NewShard(stack *Node, config *miner.Config, chainID int, addrInfo *utils.AddressInfo, clientCnt int) (*Shard, error) {
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

	bc, err := core.NewBlockChain(chainDb, nil, chainConfig, addrInfo)
	if err != nil {
		log.Error("NewShard NewBlockChain err")
		return nil, err
	}

	pool := core.NewTxPool(bc)

	headerCh := make(chan struct{})
	m := miner.NewMiner(config, bc, pool, headerCh)

	stack.chainID = chainID
	stack.addr = NodeTable[chainID]

	log.Info("NewShard", "chainID=", chainID, "addr=", stack.addr)

	miningShard := &Shard{
		chainDb:       chainDb,
		blockchain:    bc,
		txPool:        pool,
		miner:         m,
		leader:        stack,
		headerCh:      headerCh,
		exitCh:        make(chan struct{}),
		injectNotDone: int32(clientCnt),
		connMap:       make(map[string]net.Conn),
	}
	// 启动监听
	// go miningShard.TcpListen()

	// 废弃：main初始化数据集
	// miningShard.txPool.GetETHData(filepath)

	// 废弃：main注入
	// go miningShard.txPool.InjectTX(config.InjectSpeed)

	// 出块时发送数据
	miningShard.wg.Add(1)
	go miningShard.sendLoop()

	return miningShard, nil
}

func (shard *Shard) SetMessageHub(hub core.MessageHub) {
	shard.miner.SetMessageHub(hub)
}

func (shard *Shard) GetChainID() int {
	return shard.leader.chainID
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

// func (shard *Shard) InjectSingleTX(tx *core.Transaction) {
// 	shard.txPool.AddTx(tx)
// }

/**
 * main 实现 注入，这里直接添加到 txpool
 */
func (shard *Shard) InjectTXs(txs []*core.Transaction) {
	shard.txPool.AddTxs(txs)
}

func (shard *Shard) SetInjectTXDone() {
	atomic.AddInt32(&shard.injectNotDone, -1)
}

// func (shard *Shard) CanStop() bool {
// 	if shard.injectDone && shard.txPool.Empty() { // worker.commit 会继续执行最后拿出来的交易，然后才进行下一次 select
// 		return true
// 	}
// 	return false
// }

/* 交易注入完成即可停止 */
func (shard *Shard) CanStopV2() bool {
	return shard.injectNotDone == 0
}

// /* 废弃该函数， 要考虑空后又发送 cross-shard TX */
// func (shard *Shard) TryStop() {
// 	for {
// 		if shard.injectDone && shard.txPool.Empty() { // worker.commit 会继续执行最后拿出来的交易，然后才进行下一次 select
// 			shard.CloseMining()
// 			break
// 		}
// 		time.Sleep(3 * time.Millisecond)
// 	}
// }

func (shard *Shard) sendLoop() {
	defer shard.wg.Done()
	for {
		select {
		case <-shard.headerCh:
			// /* 广播 header 到其他分片 */
			// for destChainID, destAddr := range NodeTable {
			// 	if destChainID == shard.leader.chainID {
			// 		continue
			// 	}
			// 	header := shard.blockchain.CurrentBlock().Header()
			// 	mHeader, err := json.Marshal(header)
			// 	if err != nil {
			// 		log.Error("Encode header err:" + err.Error())
			// 	}
			// 	// fmt.Printf("Shard %d 正在向 分片%d 的主节点发送 header\n", shard.leader.chainID, destChainID)
			// 	message := jointMessage(cHeader, mHeader)
			// 	go shard.TcpDial(message, destAddr)
			// }

		case <-shard.exitCh:
			log.Info("shard sendLoop close..")
			return
		}

	}
}

func (s *Shard) StartMining() {
	log.Info("shard start mining ..", "shardID", s.GetChainID())
	go s.miner.Start()
}

func (s *Shard) StopMining() {
	log.Info("shard stop mining ..", "shardID", s.GetChainID())
	s.miner.Stop()
}

func (s *Shard) CloseMining() {
	s.miner.Close()
}

func (s *Shard) Close() {
	log.Debug("closing shard", "shardID", s.GetChainID())
	// 必须先关闭miner的loop，再关闭s.exitch，否则可能会阻塞在headerch。
	s.CloseMining()
	close(s.exitCh)
	s.wg.Wait()
	log.Debug("shard has been closed", "shardID", s.GetChainID())
}

func (s *Shard) TXpool() *core.TxPool {
	return s.txPool
}
