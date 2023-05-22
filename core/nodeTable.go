package core

import (
	"fmt"
	"go-w3chain/log"
)

var (
	NodeTable     map[int]*NodeConfig = make(map[int]*NodeConfig)
	host          string              = "127.0.0.1"
	availablePort int                 = 20000
)

func NewNodeConfig(nodeID int) *NodeConfig {
	// todo: 检测端口是否被占用
	config := &NodeConfig{
		Name:   fmt.Sprintf("node%d", nodeID),
		WSHost: host,
		WSPort: availablePort,
	}

	availablePort += 1
	NodeTable[nodeID] = config

	return config
}

func PrintNodeTable() {
	for k, v := range NodeTable {
		log.Info("NodeTable:", "nodeID", k, "addr", v)
	}
}
