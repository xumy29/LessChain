package shard

import (
	// "fmt"
)

const (
	DefaultWSHost      = "localhost" // Default host interface for the websocket RPC server
	DefaultWSPort      = 8546        // Default TCP port for the websocket RPC server
)


// DefaultConfig contains reasonable default settings.
var DefaultNodeConfig = NodeConfig{
	DataDir:             DefaultDataDir(),
	WSPort:              DefaultWSPort,
}