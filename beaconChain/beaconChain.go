package beaconChain

import (
	"go-w3chain/core"
	"go-w3chain/log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type TimeBeacon struct {
	Height     uint64      `json:"height" gencodec:"required"`
	ShardID    uint32      `json:"shardID" gencodec:"required"`
	BlockHash  common.Hash `json:"blockHash" gencodec:"required"`
	TxHash     common.Hash `json:"txHash" gencodec:"required"`
	StatusHash common.Hash `json:"statusHash" gencodec:"required"`
}

/* Hash returns the keccak256 hash of its RLP encoding. */
func (tb *TimeBeacon) Hash() common.Hash {
	hash, err := core.RlpHash(tb)
	if err != nil {
		log.Error("time beacon hash fail.", "err", err)
	}
	return hash
}

type SignedTB struct {
	*TimeBeacon
	Sigs [][]byte
}

type ConfirmedTB struct {
	*TimeBeacon
	ConfirmTime uint64
	/* 信标链上包含该信标的区块的高度，注意不是分片自身的区块高度 */
	ConfirmHeight uint64
}

type BeaconChain struct {
	messageHub core.MessageHub

	/* key是分片ID，value是该分片每个高度区块的信标 */
	tbs  map[int][]*ConfirmedTB
	lock sync.Mutex
	/* 已提交到信标链但还未被信标链打包 */
	tbs_new           map[int][]*SignedTB
	lock_new          sync.Mutex
	blockIntervalSecs int
	height            uint64
	stopCh            chan struct{}
	wg                sync.WaitGroup

	contract *Contract
}

/** 新建一条信标链
 * required 表示一个信标需要收到的多签名最小数量
 */
func NewTBChain(blockIntervalSecs, shardNum int, required int) *BeaconChain {
	tbChain := &BeaconChain{
		tbs:               make(map[int][]*ConfirmedTB),
		tbs_new:           make(map[int][]*SignedTB),
		blockIntervalSecs: blockIntervalSecs,
		height:            0,
		stopCh:            make(chan struct{}),
		contract:          NewContract(shardNum, required),
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

func (tbChain *BeaconChain) AddGenesisTB(signedTb *SignedTB) {
	tbChain.lock.Lock()
	defer tbChain.lock.Unlock()
	tb := signedTb.TimeBeacon
	tbs_shard := tbChain.tbs[int(tb.ShardID)]
	if tb.Height != uint64(len(tbs_shard)) {
		log.Warn("Could not add time beacon because the height didn't match!", "expected", len(tbs_shard), "got", tb.Height)
	}
	confirmedTb := &ConfirmedTB{
		TimeBeacon:    tb,
		ConfirmTime:   uint64(time.Now().Unix()),
		ConfirmHeight: 0,
	}
	tbChain.tbs[int(tb.ShardID)] = append(tbChain.tbs[int(tb.ShardID)], confirmedTb)
	log.Debug("AddGenesisTimeBeacon", "info", tb)

}

/** 调用这个函数，相当于在信标链上发起一笔交易
 * tb会被暂时存下，等待信标链打包时处理
 * 信标链打包时，会调用合约验证tb的多签名合法性，验证通过才会打包该交易，即确认该信标
 */
func (tbChain *BeaconChain) AddTimeBeacon(tb *SignedTB) {
	if tb.Height == 0 {
		tbChain.AddGenesisTB(tb)
		return
	}
	tbChain.lock_new.Lock()
	defer tbChain.lock_new.Unlock()
	tbs_shard := tbChain.tbs[int(tb.ShardID)]
	tbs_shard_new := tbChain.tbs_new[int(tb.ShardID)]
	if tb.Height != uint64(len(tbs_shard)+len(tbs_shard_new)) {
		log.Warn("Could not add time beacon because the height didn't match!", "expected", len(tbs_shard)+len(tbs_shard_new), "got", tb.Height)
	}
	tbChain.tbs_new[int(tb.ShardID)] = append(tbChain.tbs_new[int(tb.ShardID)], tb)
	log.Debug("AddTimeBeacon", "info", tb)
}

func (tbChain *BeaconChain) GetTimeBeacon(shardID int, height uint64) *ConfirmedTB {
	tbs_shard := tbChain.tbs[shardID]
	if height >= uint64(len(tbs_shard)) {
		log.Warn("Could not get the time beacon because the requested height was larger than existing confirmed height!", "requested", height, "max height", len(tbs_shard))
		return nil
	}
	return tbs_shard[height]
}
