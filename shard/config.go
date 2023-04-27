package shard

import (
	"os"
	"path/filepath"
	// "go-w3chain/utils"
)

const (
	datadir                = ".w3chain"
	datadirDefaultKeyStore = "keystore"         // Path within the datadir to the keystore
	clientIdentifier       = "shard1-Committee" // Client identifier to advertise over the network
)

func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := os.Getenv("HOME")
	return filepath.Join(home, datadir)

}

func KeyDirConfig() string {
	keydir := filepath.Join(DefaultDataDir(), datadirDefaultKeyStore)
	return keydir
}

type NodeConfig struct {
	Name    string `toml:"-"`
	DataDir string

	WSHost string
	WSPort int `toml:",omitempty"`
}

type shardConfig struct {
	Node NodeConfig
}

func defaultNodeConfig() NodeConfig {
	cfg := DefaultNodeConfig
	cfg.Name = clientIdentifier
	return cfg
}

func MakeConfigNode(name string) (*Node, shardConfig) {
	// Load defaults.
	cfg := shardConfig{
		Node: defaultNodeConfig(),
	}
	if name != "" {
		cfg.Node.DataDir = filepath.Join(DefaultDataDir(), name)
	}

	stack := New(&cfg.Node)

	return stack, cfg

}

func getKeyStoreDir(conf *NodeConfig) string {
	keydir := filepath.Join(conf.DataDir, datadirDefaultKeyStore)
	return keydir
}

// func makeConfigShard() (*Node, shardConfig){
// 	return makeConfigNode()
// }
