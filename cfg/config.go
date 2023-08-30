package cfg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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
	Height2Rollback          int    `json:"Height2Rollback"`
	Height2Reconfig          int    `json:"Height2Reconfig"`
	Height2Confirm           int    `json:"Height2Confirm"`
	MaxBlockTXSize           int    `json:"MaxBlockTXSize"`
	DatasetDir               string `json:"DatasetDir"`
	ShardSize                int    `json:"ShardSize"`
	TbchainBlockIntervalSecs int    `json:"TbchainBlockIntervalSecs"`
	MultiSignRequiredNum     int    `json:"MultiSignRequiredNum"`
	BeaconChainMode          int    `json:"BeaconChainMode"`
	BeaconChainID            int    `json:"BeaconChainID"`
	BeaconChainPort          int    `json:"BeaconChainPort"`
	ExitMode                 int    `json:"ExitMode"`
	ReconfigTime             int    `json:"ReconfigTime"`
}

var (
	defaultCfg *Cfg = nil
)

func DefaultCfg(cfgFilename string) *Cfg {
	if defaultCfg == nil {
		defaultCfg = ReadCfg(cfgFilename)
	}
	return defaultCfg
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
