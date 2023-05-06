package shard

import (
	"fmt"
	"go-w3chain/log"
)

var (
	NodeTable map[int]string
)

/*
 * 设置分片leader节点的ip和端口
 */
func SetNodeTable(shardNum int) {
	NodeTable = make(map[int]string)
	for i := 0; i < shardNum; i++ {
		// LeaderPort := 30000 + i
		LeaderPort := 20000 + i
		LeaderAddress := fmt.Sprintf("127.0.0.1:%d", LeaderPort)
		NodeTable[i] = LeaderAddress
	}
}

func PrintNodeTable() {
	for k, v := range NodeTable {
		log.Info("NodeTable:", "shardID", k, "LeaderAddr", v)
	}
}
