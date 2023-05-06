package committee

import (
	"go-w3chain/core"
	"go-w3chain/log"
)

type Committee struct {
	shardID    uint64
	config     *core.MinerConfig
	messageHub core.MessageHub
	worker     *worker
}

func NewCommittee(shardID uint64, config *core.MinerConfig) *Committee {
	com := &Committee{
		shardID: shardID,
		config:  config,
		worker:  newWorker(config, shardID),
	}
	log.Info("NewCommittee", "shardID", shardID)

	return com
}

func (com *Committee) SetMessageHub(hub core.MessageHub) {
	com.worker.SetMessageHub(hub)
}

func (com *Committee) Start() {
	com.worker.start()
}

func (com *Committee) Close() {
	com.worker.close()
}
