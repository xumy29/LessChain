package controller

import (
	"fmt"
	"go-w3chain/client"
	"go-w3chain/committee"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"go-w3chain/shard"
	"math"
	"time"
)

var shards []*shard.Shard
var committees []*committee.Committee
var clients []*client.Client

func newClients(rollbackSecs, shardNum int) {
	for cid := 0; cid < len(clients); cid++ {
		clients[cid] = client.NewClient(cid, rollbackSecs, shardNum)
		log.Info("NewClient", "Info", clients[cid])
	}
}

func newShards(shardNum int) {
	for shardID := 0; shardID < shardNum; shardID++ {
		databaseDir := fmt.Sprint("shard", shardID)
		stack, _ := shard.MakeConfigNode(databaseDir)
		shard, err := shard.NewShard(stack, shardID, len(clients))
		if err != nil {
			log.Error("NewShard failed", "err:", err)
		}
		// shards = append(shards, shard)
		shards[shardID] = shard
	}
}

func newCommittees(shardNum int, config *core.MinerConfig) {
	for shardID := 0; shardID < shardNum; shardID++ {
		com := committee.NewCommittee(uint64(shardID), config)
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

/**
 * 循环判断各分片和委员会能否停止, 若能则停止
 * 循环打印交易总执行进度
 */
func closeShardsAndCommittees(recommitIntervalSecs, logProgressInterval int, isLogProgress bool) {
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
		isInjectDone := false
		for _, shard := range shards {
			if shard.CanStopV2() {
				isInjectDone = true
				break
			}
		}
		if isInjectDone {
			for _, com := range committees {
				com.Close()
			}
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

func stopClients() {
	for _, c := range clients {
		c.Stop()
	}
}

func GetClients() []*client.Client {
	return clients
}

func GetShards() []*shard.Shard {
	return shards
}
