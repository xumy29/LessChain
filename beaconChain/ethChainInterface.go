package beaconChain

import (
	"go-w3chain/core"
	"go-w3chain/eth_chain"
	"go-w3chain/log"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	genesisTBs map[uint32]*eth_chain.ContractTB = make(map[uint32]*eth_chain.ContractTB)
	// 每个委员会指配一个client，client与以太坊私链交互，获取gasPrcie、nonce等链与账户信息
	client *ethclient.Client
	// 缓存websocket返回的事件（代表确认信标），最多缓存100个已确认的信标
	eventChannel chan *eth_chain.Event = make(chan *eth_chain.Event, 100)
)

func (tbChain *BeaconChain) HandleBooterSendContract(data *core.BooterSendContract) {
	tbChain.contractAddr = data.Addr
	contractABI, err := abi.JSON(strings.NewReader(eth_chain.MyContractABI()))
	if err != nil {
		log.Error("get contracy abi fail", "err", err)
	}
	tbChain.contractAbi = &contractABI

	go eth_chain.SubscribeEvents(tbChain.cfg.Port, tbChain.contractAddr, eventChannel)
}

func (tbChain *BeaconChain) GetEthChainLatestBlockHash(comID uint32) (common.Hash, uint64) {
	client := tbChain.getEthClient()
	return eth_chain.GetLatestBlockHash(client)
}

// func (tbChain *BeaconChain) GetEthChainBlockHash(shardID uint32, height uint64) common.Hash {
// 	client := tbChain.getEthClient()
// 	return eth_chain.GetBlockHash(client, height)
// }

func (tbChain *BeaconChain) AddTimeBeacon2EthChain(signedtb *core.SignedTB) {
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
		tbChain.AddEthChainGenesisTB(contractTB)
	} else {
		client := tbChain.getEthClient()
		err := eth_chain.AddTB(client, tbChain.contractAddr,
			tbChain.contractAbi, tbChain.mode, contractTB, signedtb.Sigs, signedtb.Vrfs,
			signedtb.SeedHeight, signedtb.Signers, tbChain.cfg.ChainId)
		if err != nil {
			log.Error("eth_chain.AddTB err", "err", err)
		}
	}
	log.Debug("AddTbTXSent", "info", signedtb)
}

func (tbChain *BeaconChain) AdjustEthChainRecordedAddrs(addrs []common.Address, vrfs [][]byte, seedHeight uint64, shardID uint32) {
	client := tbChain.getEthClient()
	err := eth_chain.AdjustRecordedAddrs(client, tbChain.contractAddr,
		tbChain.contractAbi, tbChain.mode, shardID, addrs, vrfs, seedHeight, tbChain.cfg.ChainId)
	if err != nil {
		log.Error("eth_chain.AdjustRecordedAddrs err", "err", err)
	}
	log.Debug("AdjustAddrsTXSent", "shardID", shardID, "seedHeight", seedHeight)
}

func (tbChain *BeaconChain) generateEthChainBlock() *TBBlock {
	tbs_new := make(map[uint32][]*core.TimeBeacon)
	for {
		if len(eventChannel) == 0 {
			break
		}

		event := <-eventChannel
		tb := &core.TimeBeacon{
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

	tbChain.tbs_new = make(map[int][]*core.SignedTB)

	log.Debug("TBchainGenerateBlock", "info", block)
	return block
}

func (tbChain *BeaconChain) AddEthChainGenesisTB(tb *eth_chain.ContractTB) (common.Address, *abi.ABI) {
	genesisTBs[tb.ShardID] = tb
	if len(genesisTBs) == tbChain.shardNum {
		// 转化为数组形式
		tbs := make([]eth_chain.ContractTB, tbChain.shardNum)
		for shardID, tb := range genesisTBs {
			tbs[shardID] = *tb
		}

		tbChain.deployContract(tbs)

		// go eth_chain.SubscribeEvents(tbChain.cfg.Port, tbChain.contractAddr, eventChannel)
		return tbChain.contractAddr, tbChain.contractAbi
	}
	return common.Address{}, nil
}

func (tbChain *BeaconChain) deployContract(genesisTBs []eth_chain.ContractTB) {
	// 创建合约，各分片创世区块作为构造函数的参数
	var err error
	client := tbChain.getEthClient()

	tbChain.contractAddr, tbChain.contractAbi, _, err = eth_chain.DeployContract(client,
		tbChain.mode, genesisTBs,
		uint32(tbChain.cfg.MultiSignRequiredNum),
		uint32(tbChain.shardNum),
		tbChain.addrs,
		tbChain.cfg.ChainId)
	if err != nil {
		log.Error("error occurs during deploying contract.", "err", err)
		panic(err)
	}
}

func (tbChain *BeaconChain) getEthClient() *ethclient.Client {
	if client == nil {
		var err error
		client, err = eth_chain.Connect(tbChain.cfg.Port)
		if err != nil {
			log.Error("could not connect to eth chain!", "err", err)
			panic(err)
		}
	}

	return client
}
