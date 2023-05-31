package controller

import (
	"fmt"
	"go-w3chain/beaconChain"
	"go-w3chain/client"
	"go-w3chain/committee"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"go-w3chain/shard"
	"math"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var shards []*shard.Shard
var committees []*committee.Committee
var clients []*client.Client
var nodes []*core.Node
var tbChain *beaconChain.BeaconChain

func newClients(rollbackHeight, shardNum int) {
	for cid := 0; cid < len(clients); cid++ {
		clients[cid] = client.NewClient(cid, rollbackHeight, shardNum)
		log.Info("NewClient", "Info", clients[cid])
	}
}

func newShards(shardNum int, shardSize int) {
	for shardID := 0; shardID < shardNum; shardID++ {
		shard, err := shard.NewShard(nodes[shardID*shardSize:(shardID+1)*shardSize], shardID)
		if err != nil {
			log.Error("NewShard failed", "err:", err)
		}
		// shards = append(shards, shard)
		shards[shardID] = shard
	}
}

/* 创建所有节点，并划分到不同分片中 */
func newNodes(shardNum, shardSize int) []*core.Node {
	nodes = make([]*core.Node, shardNum*shardSize)
	for shardID := 0; shardID < shardNum; shardID++ {
		shardDataDir := fmt.Sprint("shard", shardID)
		shardDataDir = filepath.Join(core.DefaultDataDir(), shardDataDir)
		for j := 0; j < shardSize; j++ {
			nodeID := shardID*shardSize + j
			config := core.NewNodeConfig(nodeID)
			nodes[nodeID] = core.NewNode(config, shardDataDir, shardID, nodeID)
		}
	}

	return nodes
}

func newCommittees(shardNum, shardSize int, config *core.MinerConfig) {
	for shardID := 0; shardID < shardNum; shardID++ {
		com := committee.NewCommittee(uint64(shardID), len(clients), nodes[shardID*shardSize:(shardID+1)*shardSize], config)
		committees[shardID] = com
	}
}

/* 启动所有分片 */
func startShards() {
	for _, shard := range shards {
		shard.AddGenesisTB()
	}
}

/* 启动所有委员会 */
func startCommittees() {
	for _, com := range committees {
		com.Start()
	}
}

func startClients(injectSpeed int, recommitIntervalSecs int, addrTable map[common.Address]int) {
	for _, c := range clients {
		c.Start(injectSpeed, recommitIntervalSecs, addrTable)
	}
}

/**
 * 循环判断各客户端和委员会能否停止, 若能则停止
 * 循环打印交易总执行进度
 */
func closeCommittees(recommitIntervalSecs, logProgressInterval int, isLogProgress bool) {
	log.Info("Monitor txpools and try to stop shards")
	sleepSecs := int(math.Ceil(float64(recommitIntervalSecs) / 2))
	iterNum := int(math.Ceil(float64(logProgressInterval) / float64(sleepSecs)))
	iter := 0
	if isLogProgress {
		log.Info("Set logProgressInterval(secs)", "iterNum*sleepSecs", iterNum*sleepSecs)
	} else {
		log.Info("Set log progress false")
	}
	for {
		isInjectDone := true
		for _, c := range clients {
			isInjectDone = isInjectDone && c.CanStopV1()
		}
		for _, com := range committees {
			isInjectDone = isInjectDone && com.CanStopV1()
		}
		if isInjectDone {
			for _, com := range committees {
				com.Close()
			}
			stopClients()
			break
		}
		// 每出块间隔的一半时间打印一次进度
		time.Sleep(time.Duration(sleepSecs) * time.Second)
		/* 打印进度 */
		if isLogProgress {
			iter++
			if iter == iterNum {
				result.GetPercentage()
				iter = 0
			}
		}
	}
}

func closeNodes() {
	for _, n := range nodes {
		n.Close()
	}
}

func stopClients() {
	for _, c := range clients {
		c.Stop()
	}
}

func stopTBChain() {
	tbChain.Close()
}
