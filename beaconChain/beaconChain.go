package beaconChain

import (
	"go-w3chain/core"
	"go-w3chain/log"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type TimeBeacon struct {
	Height      uint64
	ShardID     int
	BlockHash   common.Hash
	TxHash      common.Hash
	StatusHash  common.Hash
	ConfirmTime uint64
	/* 信标链上包含该信标的区块的高度，注意不是分片自身的区块高度 */
	ConfirmHeight uint64
}

type BeaconChain struct {
	messageHub core.MessageHub

	/* key是分片ID，value是该分片每个高度区块的信标 */
	tbs  map[int][]*TimeBeacon
	lock sync.Mutex
	/* 已提交到信标链但还未被信标链打包 */
	tbs_new           map[int][]*TimeBeacon
	lock_new          sync.Mutex
	blockIntervalSecs int
	height            uint64
	stopCh            chan struct{}
	wg                sync.WaitGroup
}

func NewTBChain(blockIntervalSecs int) *BeaconChain {
	tbChain := &BeaconChain{
		tbs:               make(map[int][]*TimeBeacon),
		tbs_new:           make(map[int][]*TimeBeacon),
		blockIntervalSecs: blockIntervalSecs,
		height:            0,
		stopCh:            make(chan struct{}),
	}
	log.Info("NewTBChain")
	tbChain.wg.Add(1)
	go tbChain.loop()
	return tbChain
}

func (tbChain *BeaconChain) Close() {
	close(tbChain.stopCh)
	tbChain.wg.Wait()
	log.Info("tbchain close")
}

func (tbChain *BeaconChain) SetMessageHub(hub core.MessageHub) {
	tbChain.messageHub = hub
}

func (tbChain *BeaconChain) AddGenesisTB(tb *TimeBeacon) {
	tbChain.lock.Lock()
	defer tbChain.lock.Unlock()
	tbs_shard := tbChain.tbs[tb.ShardID]
	if tb.Height != uint64(len(tbs_shard)) {
		log.Warn("Could not add time beacon because the height didn't match!", "expected", len(tbs_shard), "got", tb.Height)
	}
	tbChain.tbs[tb.ShardID] = append(tbChain.tbs[tb.ShardID], tb)
	log.Debug("AddGenesisTimeBeacon", "info", tb)

}

func (tbChain *BeaconChain) AddTimeBeacon(tb *TimeBeacon) {
	if tb.Height == 0 {
		tbChain.AddGenesisTB(tb)
		return
	}
	tbChain.lock_new.Lock()
	defer tbChain.lock_new.Unlock()
	tbs_shard := tbChain.tbs[tb.ShardID]
	tbs_shard_new := tbChain.tbs_new[tb.ShardID]
	if tb.Height != uint64(len(tbs_shard)+len(tbs_shard_new)) {
		log.Warn("Could not add time beacon because the height didn't match!", "expected", len(tbs_shard)+len(tbs_shard_new), "got", tb.Height)
	}
	tbChain.tbs_new[tb.ShardID] = append(tbChain.tbs_new[tb.ShardID], tb)
	log.Debug("AddTimeBeacon", "info", tb)
}

func (tbChain *BeaconChain) GetTimeBeacon(shardID int, height uint64) *TimeBeacon {
	tbs_shard := tbChain.tbs[shardID]
	if height >= uint64(len(tbs_shard)) {
		log.Warn("Could not get the time beacon because the requested height was larger than existing confirmed height!", "requested", height, "max height", len(tbs_shard))
		return nil
	}
	return tbs_shard[height]
}
