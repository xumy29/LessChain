package beaconChain

import (
	"go-w3chain/core"
	"go-w3chain/log"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type TimeBeacon struct {
	Height     uint64
	ShardID    uint64
	BlockHash  common.Hash
	TxHash     common.Hash
	StatusHash common.Hash
}

type BeaconChain struct {
	messageHub core.MessageHub

	/* key是分片ID，value是该分片每个高度区块的信标 */
	tbs  map[uint64][]*TimeBeacon
	lock sync.Mutex
}

func NewTBChain() *BeaconChain {
	tbChain := &BeaconChain{
		tbs: make(map[uint64][]*TimeBeacon),
	}
	log.Info("NewTBChain")
	return tbChain
}

func (tbChain *BeaconChain) SetMessageHub(hub core.MessageHub) {
	tbChain.messageHub = hub
}

func (tbChain *BeaconChain) AddTimeBeacon(tb *TimeBeacon) {
	tbChain.lock.Lock()
	defer tbChain.lock.Unlock()
	tbs_shard := tbChain.tbs[tb.ShardID]
	if tb.Height != uint64(len(tbs_shard)) {
		log.Warn("Could not add time beacon because the height didn't match!", "expected", len(tbs_shard), "got", tb.Height)
	}
	tbChain.tbs[tb.ShardID] = append(tbChain.tbs[tb.ShardID], tb)
	log.Debug("AddTimeBeacon", "info", tb)
}

func (tbChain *BeaconChain) GetTimeBeacon(shardID, height uint64) *TimeBeacon {
	tbs_shard := tbChain.tbs[shardID]
	if height > uint64(len(tbs_shard)) {
		log.Warn("Could not get the time beacon because the requested height was larger than existing max height!", "requested", height, "max height", len(tbs_shard))
		return nil
	}
	return tbs_shard[height]
}
