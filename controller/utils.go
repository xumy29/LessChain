package controller

import (
	"fmt"
	"go-w3chain/client"
	"go-w3chain/log"
	"go-w3chain/miner"
	"go-w3chain/result"
	"go-w3chain/shard"
	"go-w3chain/utils"
	"math"
	"time"
)

var shards []*shard.Shard
var clients []*client.Client

func newClients(rollbackSecs, shardNum int) {
	for cid := 0; cid < len(clients); cid++ {
		clients[cid] = client.NewClient(cid, rollbackSecs, shardNum)
		log.Info("NewClient", "Info", clients[cid])
	}
}

func newShards(shardNum int, config *miner.Config, addrInfo *utils.AddressInfo) {
	for shardID := 0; shardID < shardNum; shardID++ {
		databaseDir := fmt.Sprint("shard", shardID)
		stack, _ := shard.MakeConfigNode(databaseDir)
		shard, err := shard.NewShard(stack, config, shardID, addrInfo, len(clients))
		if err != nil {
			log.Error("NewShard failed", "err:", err)
		}
		// shards = append(shards, shard)
		shards[shardID] = shard
	}
}

/* 启动所有分片 */
func startShards() {
	for _, shard := range shards {
		shard.StartMining()
	}
}

/* 当交易停止注入时，停止所有分片 */
func closeShardsV2(recommitIntervalSecs, progressInterval int, isLogProgress bool) {
	log.Info("Monitor txpools and try to stop shards")
	sleepSecs := int(math.Ceil(float64(recommitIntervalSecs) / 2))
	iterNum := int(math.Ceil(float64(progressInterval) / float64(sleepSecs)))
	iter := 0
	if isLogProgress {
		log.Info("Set progressInterval(secs)", "iterNum*sleepSecs", iterNum*sleepSecs)
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
			msg := fmt.Sprintf("close shards begin .., there are %d shards to close.", len(shards))
			log.Info(msg)
			for _, shard := range shards {
				shard.Close()
			}
			// 异步关闭分片
			// var wg sync.WaitGroup
			// for i, _ := range shards {
			// 	wg.Add(1)
			// 	tmp := i
			// 	go func() {
			// 		shards[tmp].Close()
			// 		wg.Done()
			// 	}()
			// }
			// wg.Wait()
			log.Info("all shards' main routines and sub-routines(mining routines) are shut down!")
			break
			/* close shard 后 close 连接 */
			// for _, shard := range shards {
			// 	shard.StopTCPConn()
			// }
		}

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
