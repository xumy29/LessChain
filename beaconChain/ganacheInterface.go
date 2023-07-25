package beaconChain

import (
	"go-w3chain/ganache"
	"go-w3chain/log"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	genesisTBs map[uint32]*ganache.ContractTB = make(map[uint32]*ganache.ContractTB)
	// 每个委员会指配一个client，client与ganache交互，获取gasPrcie、nonce等链与账户信息
	// 并发调用client的方法或短时间内连续调用有时会出现问题，比如connection reset by peer等，应尽量避免
	clients      []*ethclient.Client
	contractAddr common.Address
	contractABI  *abi.ABI
	// 缓存websocket返回的事件（代表确认信标），最多缓存100个已确认的信标
	eventChannel chan *ganache.Event = make(chan *ganache.Event, 100)
)

func (tbChain *BeaconChain) AddTimeBeacon2GanacheChain(signedtb *SignedTB) {
	tb := signedtb.TimeBeacon
	// 将字节数组转化为字符串
	contractTB := &ganache.ContractTB{
		ShardID:    tb.ShardID,
		Height:     tb.Height,
		BlockHash:  tb.BlockHash.Hex(),
		TxHash:     tb.TxHash.Hex(),
		StatusHash: tb.StatusHash.Hex(),
	}
	if tb.Height == 0 {
		tbChain.addGanacheGenesisTB(contractTB)
	} else {
		ganache.AddTB(clients[contractTB.ShardID], contractAddr, contractABI, contractTB)
	}
}

func (tbChain *BeaconChain) generateGanacheChainBlock() *TBBlock {
	tbs_new := make(map[uint32][]*TimeBeacon)
	for {
		if len(eventChannel) == 0 {
			break
		}

		event := <-eventChannel
		tb := &TimeBeacon{
			ShardID: event.ShardID,
			Height:  event.Height,
		}
		tbs_new[tb.ShardID] = append(tbs_new[tb.ShardID], tb)
	}

	now := time.Now().Unix()
	tbChain.height += 1 // todo: 调整为真实高度

	confirmTBs := make(map[uint32][]*ConfirmedTB, 0)
	for shardID, tbs := range tbs_new {
		for _, tb := range tbs {
			confirmedTB := &ConfirmedTB{
				TimeBeacon:    tb,
				ConfirmTime:   uint64(now),
				ConfirmHeight: tbChain.height,
			}
			confirmTBs[shardID] = append(confirmTBs[shardID], confirmedTB)
		}
		tbChain.tbs[int(shardID)] = append(tbChain.tbs[int(shardID)], confirmTBs[shardID]...)
	}

	block := &TBBlock{
		Tbs:    confirmTBs,
		Time:   uint64(now),
		Height: tbChain.height,
	}

	tbChain.tbs_new = make(map[int][]*SignedTB)

	log.Debug("tbchain generate block", "info", block)
	return block
}

func (tbChain *BeaconChain) addGanacheGenesisTB(tb *ganache.ContractTB) {
	genesisTBs[tb.ShardID] = tb
	if len(genesisTBs) == tbChain.shardNum {
		// 转化为数组形式
		tbs := make([]ganache.ContractTB, tbChain.shardNum)
		for shardID, tb := range genesisTBs {
			tbs[shardID] = *tb
		}

		ganache.SetChainID(tbChain.chainID)
		tbChain.deployContract(tbs)

		go ganache.SubscribeEvents(tbChain.chainPort, contractAddr, eventChannel)

	}
}

func (tbChain *BeaconChain) deployContract(genesisTBs []ganache.ContractTB) {
	// 创建合约，各分片创世区块作为构造函数的参数
	var err error
	for i := 0; i < tbChain.shardNum; i++ {
		client, err := ganache.Connect(tbChain.chainPort)
		if err != nil {
			log.Error("could not connect to ganache chain!", "err", err)
			panic(err)
		}
		clients = append(clients, client)
	}

	contractAddr, contractABI, _, err = ganache.DeployContract(clients[0], genesisTBs)
	if err != nil {
		log.Error("error occurs during deploying contract.", "err", err)
		panic(err)
	}
}
