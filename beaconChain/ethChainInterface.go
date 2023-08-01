package beaconChain

import (
	"go-w3chain/eth_chain"
	"go-w3chain/log"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	genesisTBs map[uint32]*eth_chain.ContractTB = make(map[uint32]*eth_chain.ContractTB)
	// 每个委员会指配一个client，client与以太坊私链交互，获取gasPrcie、nonce等链与账户信息
	clients      []*ethclient.Client
	contractAddr common.Address
	contractABI  *abi.ABI
	// 缓存websocket返回的事件（代表确认信标），最多缓存100个已确认的信标
	eventChannel chan *eth_chain.Event = make(chan *eth_chain.Event, 100)
)

func (tbChain *BeaconChain) AddTimeBeacon2EthChain(signedtb *SignedTB) {
	tb := signedtb.TimeBeacon
	// 转化为合约中的结构（目前两结构的成员变量是完全相同的）
	contractTB := &eth_chain.ContractTB{
		ShardID:    tb.ShardID,
		Height:     tb.Height,
		BlockHash:  tb.BlockHash,
		TxHash:     tb.TxHash,
		StatusHash: tb.StatusHash,
	}
	if tb.Height == 0 {
		tbChain.addEthChainGenesisTB(contractTB)
	} else {
		eth_chain.AddTB(clients[contractTB.ShardID], contractAddr, contractABI, tbChain.mode, contractTB, signedtb.Sigs, signedtb.Signers)
	}
	log.Debug("AddTimeBeacon", "info", signedtb)
}

func (tbChain *BeaconChain) generateEthChainBlock() *TBBlock {
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

	confirmTBs := make([][]*ConfirmedTB, tbChain.shardNum)
	for shardID, tbs := range tbs_new {
		for _, tb := range tbs {
			confirmedTB := &ConfirmedTB{
				TimeBeacon:    *tb,
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

func (tbChain *BeaconChain) addEthChainGenesisTB(tb *eth_chain.ContractTB) {
	genesisTBs[tb.ShardID] = tb
	if len(genesisTBs) == tbChain.shardNum {
		// 转化为数组形式
		tbs := make([]eth_chain.ContractTB, tbChain.shardNum)
		for shardID, tb := range genesisTBs {
			tbs[shardID] = *tb
		}

		eth_chain.SetChainID(tbChain.chainID)
		tbChain.deployContract(tbs)

		go eth_chain.SubscribeEvents(tbChain.chainPort, contractAddr, eventChannel)

	}
}

func (tbChain *BeaconChain) deployContract(genesisTBs []eth_chain.ContractTB) {
	// 创建合约，各分片创世区块作为构造函数的参数
	var err error
	for i := 0; i < tbChain.shardNum; i++ {
		client, err := eth_chain.Connect(tbChain.chainPort)
		if err != nil {
			log.Error("could not connect to eth chain!", "err", err)
			panic(err)
		}
		clients = append(clients, client)
	}

	contractAddr, contractABI, _, err = eth_chain.DeployContract(clients[0], tbChain.mode, genesisTBs, tbChain.required_sig_cnt)
	if err != nil {
		log.Error("error occurs during deploying contract.", "err", err)
		panic(err)
	}
}
