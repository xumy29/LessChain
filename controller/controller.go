package controller

// GO111MODULE=on go run main.go

import (
	"encoding/json"
	"fmt"
	beaconchain "go-w3chain/beaconChain"
	"go-w3chain/client"
	"go-w3chain/data"
	"go-w3chain/log"
	"go-w3chain/messageHub"
	"go-w3chain/miner"

	// "go-w3chain/miner"
	"go-w3chain/result"
	"go-w3chain/shard"
	"io/ioutil"
	"os"
	"time"
)

type Cfg struct {
	LogLevel                   int    `json:"LogLevel"`
	LogFile                    string `json:"LogFile"`
	ProgressInterval           int    `json:"ProgressInterval"`
	IsProgressBar              bool   `json:"IsProgressBar"`
	IsLogProgress              bool   `json:"IsLogProgress"`
	ClientNum                  int    `json:"ClientNum"`
	ShardNum                   int    `json:"ShardNum"`
	MaxTxNum                   int    `json:"MaxTxNum"`
	InjectSpeed                int    `json:"InjectSpeed"`
	RecommitIntervalSecs       int    `json:"RecommitInterval"`
	RecommitIntervals2Rollback int    `json:"RecommitIntervals2Rollback"`
	MaxBlockTXSize             int    `json:"MaxBlockTXSize"`
	DatasetDir                 string `json:"datasetDir"`
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
	// vitalLogFile := ""     // 存储最必要的信息，忽略每个交易的完成状态
	ProgressInterval := cfg.ProgressInterval
	IsProgressBar := cfg.IsProgressBar
	IsLogProgress := cfg.IsLogProgress
	clientNum := cfg.ClientNum
	shardNum := cfg.ShardNum
	maxTxNum := cfg.MaxTxNum
	injectSpeed := cfg.InjectSpeed
	recommitIntervalSecs := cfg.RecommitIntervalSecs
	recommitInterval := time.Duration(recommitIntervalSecs) * time.Second
	rollbackSecs := cfg.RecommitIntervals2Rollback * cfg.RecommitIntervalSecs
	maxBlockTXSize := cfg.MaxBlockTXSize
	datasetDir := cfg.DatasetDir

	/* 设置 是否使用 progressbar */
	result.SetIsProgressBar(IsProgressBar)

	/* 日志 */
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
	// fmt.Println("vital log file:", vitalLogFile)
	result.SetcsvFilename(LogFile)
	log.SetLogInfo(log.Lvl(logLevel), LogFile)
	/* 打印超参数 */
	log.Info(fmt.Sprintf("%#v", cfg))

	/* 初始化数据集，交易分配列表 */
	// filepath := "./data/0to999999_NormalTransaction.csv"
	data.LoadETHData(datasetDir, maxTxNum)
	// 一次性注入交易到客户端
	// clientNum := 1 // 目前只支持一个客户端，未来可以扩展到多个
	clients = make([]*client.Client, clientNum)
	newClients(rollbackSecs, shardNum)
	data.SetTX2ClientTable(clientNum)
	data.InjectTX2Client(clients)
	for _, c := range clients {
		c.Print()
	}

	/* 初始化节点列表 */
	shard.SetNodeTable(shardNum)
	// shard.PrintNodeTable()

	/* 设置 addrTable */
	data.SetAddrTable(shardNum)
	/* 获取 addrInfo 信息 */
	addrInfo := data.GetAddrInfo()

	/* 初始化分片, 获取 shardaddrs，用于初始账户状态 InjectSpeed在config中无用被废弃 */
	minerConfig := &miner.Config{
		Recommit:     recommitInterval,
		MaxBlockSize: maxBlockTXSize,
		InjectSpeed:  injectSpeed,
	}
	shards = make([]*shard.Shard, shardNum) // 'shards' is delcared in utils
	newShards(shardNum, minerConfig, addrInfo)
	data.SetShardAddrs(shardNum)
	data.SetShardsInitialState(shards)

	/* 获取交易注入table*/
	data.SetTxTable(shardNum)

	tbChain := beaconchain.NewTBChain()

	/* 设置各个分片和客户端、信标链的通信渠道 */
	messageHub.Init(clients, shards, tbChain)
	/* 启动各个分片 */
	startShards()

	// /* 启动 交易注入 协程*/
	// go data.InjectTXs(injectSpeed, shards)

	/* 客户端按一定速率注入到分片中 */
	for _, c := range clients {
		go c.SendTXs(injectSpeed, shards, data.GetTxTable(), data.GetAddrTable())
		go c.CheckExpiredTXs(recommitIntervalSecs)
	}

	/* 循环判断各分片能否停止, 若能则停止；循环打印进度 */
	closeShardsV2(recommitIntervalSecs, ProgressInterval, IsLogProgress)
	/* 停止客户端的checkExpiredTXs线程 */
	stopClients()

	/* 打印交易结果 */
	// data.PrintTXs(maxTxNum)
	result.PrintTXReceipt()
	thrput, avlatency, rollbackRate, overloads := result.GetThroughtPutAndLatencyV2()
	log.Info("GetThroughtPutAndLatency", "thrput", thrput, "avlatency", avlatency, "rollbackRate", rollbackRate, "overloads", overloads)

	/* 结束 */
	log.Info(" Run finished! Bye~")
	fmt.Println(" Run finished! Bye~")
}
