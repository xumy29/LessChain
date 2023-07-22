package controller

// GO111MODULE=on go run main.go

import (
	"encoding/json"
	"fmt"
	beaconchain "go-w3chain/beaconChain"
	"go-w3chain/client"
	"go-w3chain/committee"
	"go-w3chain/core"
	"go-w3chain/data"
	"go-w3chain/log"
	"go-w3chain/messageHub"

	// "go-w3chain/miner"
	"go-w3chain/result"
	"go-w3chain/shard"
	"io/ioutil"
	"os"
	"time"
)

type Cfg struct {
	LogLevel                 int    `json:"LogLevel"`
	LogFile                  string `json:"LogFile"`
	ProgressInterval         int    `json:"ProgressInterval"`
	IsProgressBar            bool   `json:"IsProgressBar"`
	IsLogProgress            bool   `json:"IsLogProgress"`
	ClientNum                int    `json:"ClientNum"`
	ShardNum                 int    `json:"ShardNum"`
	MaxTxNum                 int    `json:"MaxTxNum"`
	InjectSpeed              int    `json:"InjectSpeed"`
	RecommitIntervalSecs     int    `json:"RecommitInterval"`
	TBChainHeight2Rollback   int    `json:"TBChainHeight2Rollback"`
	Height2Reconfig          int    `json:"Height2Reconfig"`
	MaxBlockTXSize           int    `json:"MaxBlockTXSize"`
	DatasetDir               string `json:"DatasetDir"`
	ShardSize                int    `json:"ShardSize"`
	TbchainBlockIntervalSecs int    `json:"TbchainBlockIntervalSecs"`
	MultiSignRequiredNum     int    `json:"MultiSignRequiredNum"`
	BeaconChainMode          int    `json:"BeaconChainMode"`
	BeaconChainID            int    `json:"BeaconChainID"`
}

func ReadCfg(filename string) *Cfg {
	jsonFile, err := os.Open(filename)
	if err != nil {
		fmt.Println("error opening json file")
		os.Exit(1)
	}
	defer jsonFile.Close()
	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("error reading json file")
		os.Exit(1)
	}
	var cfg Cfg
	json.Unmarshal(jsonData, &cfg)
	// log.Info(fmt.Sprintf("%#v", cfg)) // 此处还未初始化
	return &cfg
}

/** go build -o brokerChain.exe
 * brokerChain.exe -m run >> nohup.out 2>&1
 */
func Main(cfgfilename string) {
	cfg := ReadCfg(cfgfilename)

	/* 超参数 */
	logLevel := cfg.LogLevel
	LogFile := cfg.LogFile // 存储完整的日志
	ProgressInterval := cfg.ProgressInterval
	IsProgressBar := cfg.IsProgressBar
	IsLogProgress := cfg.IsLogProgress
	clientNum := cfg.ClientNum
	shardNum := cfg.ShardNum
	maxTxNum := cfg.MaxTxNum
	injectSpeed := cfg.InjectSpeed
	recommitIntervalSecs := cfg.RecommitIntervalSecs
	recommitInterval := time.Duration(recommitIntervalSecs) * time.Second
	rollbackHeight := cfg.TBChainHeight2Rollback
	/* 注意这里的height指的是信标链上区块的height，不是矿工所在分片的区块的height */
	height2Reconfig := cfg.Height2Reconfig
	maxBlockTXSize := cfg.MaxBlockTXSize
	datasetDir := cfg.DatasetDir
	/* 一个分片有多少个节点 */
	shardSize := cfg.ShardSize
	TbchainBlockIntervalSecs := cfg.TbchainBlockIntervalSecs
	MultiSignRequiredNum := cfg.MultiSignRequiredNum
	/* mode=0表示运行模拟信标链，mode=1表示运行以太坊私链 */
	beaconChainMode := cfg.BeaconChainMode
	beaconChainID := cfg.BeaconChainID

	/* 设置 是否使用 progressbar */
	result.SetIsProgressBar(IsProgressBar)

	/* 设置日志存储路径 */
	if LogFile == "" {
		if maxTxNum == -1 {
			LogFile = fmt.Sprintf("result/S%d-TXAll-Rate%d-Interval%d-BlockTX%d.log",
				shardNum, injectSpeed, recommitIntervalSecs, maxBlockTXSize)

		} else {
			LogFile = fmt.Sprintf("result/latency/S%d-TX%d-Rate%d-Interval%d-BlockTX%d.log",
				shardNum, maxTxNum, injectSpeed, recommitIntervalSecs, maxBlockTXSize)
		}

	}
	fmt.Println("log file:", LogFile)
	result.SetcsvFilename(LogFile)
	log.SetLogInfo(log.Lvl(logLevel), LogFile)

	/* 打印超参数 */
	log.Info(fmt.Sprintf("%#v", cfg))

	/* 加载数据集 */
	data.LoadETHData(datasetDir, maxTxNum)

	/* 创建一个消息中心 */
	messageHub := messageHub.NewMessageHub()

	/* 初始化客户端，分配交易到客户端 */
	clients = make([]*client.Client, clientNum)
	newClients(rollbackHeight, shardNum)

	data.SetTX2ClientTable(clientNum)
	data.InjectTX2Client(clients)
	for _, c := range clients {
		c.Print()
	}

	/* 初始化节点 */
	nodes = newNodes(shardNum, shardSize)

	/* 初始化分片，划分账户到分片，初始化分片的sender账户状态 */
	shards = make([]*shard.Shard, shardNum) // 'shards' is delcared in utils
	newShards(shardNum, shardSize)

	data.SetAddrTable(shardNum)
	data.SetShardsInitialState(shards)

	/* 初始化委员会 */
	committeeConfig := &core.CommitteeConfig{
		Recommit:             recommitInterval,
		MaxBlockSize:         maxBlockTXSize,
		InjectSpeed:          injectSpeed,
		Height2Reconfig:      height2Reconfig,
		MultiSignRequiredNum: MultiSignRequiredNum,
	}

	committees = make([]*committee.Committee, shardNum)
	newCommittees(shardNum, shardSize, committeeConfig)

	/* 初始化信标链 */
	tbChain = beaconchain.NewTBChain(beaconChainMode, beaconChainID, TbchainBlockIntervalSecs, shardNum, MultiSignRequiredNum)

	/* 设置各个分片、委员会和客户端、信标链的通信渠道 */
	messageHub.Init(clients, shards, committees, nodes, tbChain)

	/* 启动分片和委员会 */
	startShards()
	startCommittees()

	/* 客户端按一定速率将交易注入到分片中，以及开启自身的线程 */
	startClients(injectSpeed, recommitIntervalSecs, data.GetAddrTable())

	/* 循环判断各客户端和委员会能否停止, 若能则停止；循环打印进度 */
	closeCommittees(recommitIntervalSecs, ProgressInterval, IsLogProgress)
	closeNodes()

	stopTBChain()

	/* 打印交易执行结果 */
	result.PrintTXReceipt()
	thrput, avlatency, rollbackRate, overloads := result.GetThroughtPutAndLatencyV2()
	log.Info("GetThroughtPutAndLatency", "thrput", thrput, "avlatency", avlatency, "rollbackRate", rollbackRate, "overloads", overloads)

	/* 结束 */
	log.Info(" Run finished! Bye~")
	fmt.Println(" Run finished! Bye~")
}
