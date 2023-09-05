package controller

// GO111MODULE=on go run main.go

import (
	"fmt"
	beaconchain "go-w3chain/beaconChain"
	"go-w3chain/cfg"
	"go-w3chain/client"
	"go-w3chain/committee"
	"go-w3chain/core"
	"go-w3chain/data"
	"go-w3chain/log"
	"go-w3chain/messageHub"
	"go-w3chain/node"
	"go-w3chain/shard"
	"net"
	"strconv"

	// "go-w3chain/miner"
	"go-w3chain/result"
	"time"
)

func runClient(allCfg *cfg.Cfg) {
	cid := allCfg.ClientId

	client := client.NewClient(cid, allCfg.Height2Rollback, allCfg.ShardNum, allCfg.ExitMode)
	log.Info("NewClient", "Info", client)

	// 加载交易数据
	data.LoadETHData(allCfg.DatasetDir, allCfg.MaxTxNum)
	data.SetTxShardId(allCfg.ShardNum)

	// 注入交易到客户端
	data.SetTX2ClientTable(allCfg.ClientNum)
	data.InjectTX2Client(client)

	client.Print()

	// 初始化信标链接口
	beaconChainConfig := &core.BeaconChainConfig{
		Mode:                 allCfg.BeaconChainMode,
		ChainId:              allCfg.BeaconChainID,
		Port:                 allCfg.BeaconChainPort,
		BlockInterval:        allCfg.TbchainBlockIntervalSecs,
		Height2Confirm:       uint64(allCfg.Height2Confirm),
		MultiSignRequiredNum: allCfg.MultiSignRequiredNum,
	}
	tbChain = beaconchain.NewTBChain(beaconChainConfig, allCfg.ShardNum)

	/* 创建消息中心(用于客户端和信标链的交互等) */
	messageHub := messageHub.NewMessageHub()

	/* 设置各个分片、委员会和客户端、信标链的通信渠道 */
	messageHub.Init(client, nil, nil, nil, tbChain, allCfg.ShardNum)

	startClient(client, allCfg.InjectSpeed, allCfg.RecommitIntervalSecs)
	toStopClient(client, allCfg.RecommitIntervalSecs, allCfg.LogProgressInterval,
		allCfg.IsLogProgress, allCfg.ExitMode)

	stopTBChain()
	messageHub.Close()

}

func runNode(allCfg *cfg.Cfg) {
	shardId := allCfg.ShardId
	nodeId := allCfg.NodeId

	// 节点地址信息
	addr := cfg.NodeTable[uint32(shardId)][nodeId]
	host, port_str, err := net.SplitHostPort(addr)
	if err != nil {
		log.Error("invalid node address!", "addr", addr)
	}
	port, _ := strconv.ParseInt(port_str, 10, 32)
	nodeAddrConfig := &core.NodeAddrConfig{
		Name: fmt.Sprintf("S%dN%d", shardId, nodeId),
		Host: host,
		Port: int(port),
	}

	// 节点数据存储目录
	dataDir := cfg.DefaultDataDir()

	// 创建节点
	node := node.NewNode(nodeAddrConfig, dataDir, shardId, nodeId)

	// TODO：建立分片内连接

	// 创建本节点对应的分片实例，用于执行分片的操作
	shard := shard.NewShard(uint32(shardId), node)
	node.SetShard(shard)

	// 加载交易数据
	data.LoadETHData(allCfg.DatasetDir, allCfg.MaxTxNum)
	data.SetTxShardId(allCfg.ShardNum)
	// 初始化分片中的账户状态
	data.SetShardInitialAccountState(shard)

	// 创建本节点对应的委员会实例
	committeeConfig := &core.CommitteeConfig{
		RecommitTime:         time.Duration(allCfg.RecommitIntervalSecs) * time.Second,
		MaxBlockSize:         allCfg.MaxBlockTXSize,
		InjectSpeed:          allCfg.InjectSpeed,
		Height2Reconfig:      allCfg.Height2Reconfig,
		MultiSignRequiredNum: allCfg.MultiSignRequiredNum,
	}
	com := committee.NewCommittee(uint32(allCfg.ShardId), allCfg.ClientNum, node, committeeConfig)
	node.SetCommittee(com)

	// 初始化信标链接口
	beaconChainConfig := &core.BeaconChainConfig{
		Mode:                 allCfg.BeaconChainMode,
		ChainId:              allCfg.BeaconChainID,
		Port:                 allCfg.BeaconChainPort,
		BlockInterval:        allCfg.TbchainBlockIntervalSecs,
		Height2Confirm:       uint64(allCfg.Height2Confirm),
		MultiSignRequiredNum: allCfg.MultiSignRequiredNum,
	}
	tbChain = beaconchain.NewTBChain(beaconChainConfig, allCfg.ShardNum)

	/* 创建消息中心(用于委员会和信标链的交互等) */
	messageHub := messageHub.NewMessageHub()

	/* 设置各个分片、委员会和客户端、信标链的通信渠道 */
	messageHub.Init(nil, shard, com, node, tbChain, allCfg.ShardNum)

	// 启动节点对应的分片实例和委员会实例
	startShard(shard)
	startCommittee(com, nodeId)

	/* 循环打印进度；判断各客户端和委员会能否停止, 若能则停止 */
	toStopCommittee(node, allCfg.RecommitIntervalSecs, allCfg.LogProgressInterval,
		allCfg.IsLogProgress, allCfg.ExitMode)

	closeNode(node)

	stopTBChain()
	messageHub.Close()
}

func Main(cfgfilename string) {
	cfg := cfg.DefaultCfg(cfgfilename)

	/* 设置日志存储路径 */
	fmt.Println("log file:", cfg.LogFile)
	log.SetLogInfo(log.Lvl(cfg.LogLevel), cfg.LogFile)

	/* 设置 是否使用 progressbar */
	result.SetIsProgressBar(cfg.IsProgressBar)
	result.SetcsvFilename(cfg.LogFile)

	/* 打印超参数 */
	log.Info(fmt.Sprintf("%#v", cfg))

	roleType := cfg.RoleType
	switch roleType {
	case 1:
		runClient(cfg)
	case 2:
		runNode(cfg)
	default:
		log.Error("unknown roleType", "type", roleType)
	}

	/* 打印交易执行结果 */
	result.PrintTXReceipt()
	thrput, avlatency, rollbackRate, overloads := result.GetThroughtPutAndLatencyV2()
	log.Info("GetThroughtPutAndLatency", "thrput", thrput, "avlatency", avlatency, "rollbackRate", rollbackRate, "overloads", overloads)

	/* 结束 */
	log.Info(" Run finished! Bye~")
	fmt.Println(" Run finished! Bye~")
}
