package miner

import (
	"go-w3chain/core"
	"go-w3chain/log"
	"sync"
	"time"
)

// Config is the configuration parameters of mining.
type Config struct {
	Recommit     time.Duration // The time interval for miner to re-create mining work.
	MaxBlockSize int
	InjectSpeed  int
}

// Miner creates blocks and searches for proof-of-work values.
type Miner struct {
	chain *core.BlockChain
	pool  *core.TxPool

	exitCh  chan struct{}
	startCh chan struct{}
	stopCh  chan struct{}
	worker  *worker

	wg sync.WaitGroup
}

func NewMiner(config *Config, chain *core.BlockChain, pool *core.TxPool, headerCh chan<- struct{}) *Miner {
	miner := &Miner{
		chain: chain,
		pool:  pool,

		exitCh:  make(chan struct{}),
		startCh: make(chan struct{}),
		stopCh:  make(chan struct{}),

		worker: newWorker(config, chain, pool, headerCh),
	}
	miner.wg.Add(1)
	go miner.update()
	return miner

}

func (miner *Miner) SetMessageHub(hub core.MessageHub) {
	miner.worker.MessageHub = hub
}

func (miner *Miner) Start() {
	miner.startCh <- struct{}{}
}

// update keeps track of the downloader events. Please be aware that this is a one shot type of update loop.
// It's entered once and as soon as `Done` or `Failed` has been broadcasted the events are unregistered and
// the loop is exited. This to prevent a major security vuln where external parties can DOS you with blocks
// and halt your mining operation for as long as the DOS continues.
func (miner *Miner) update() {
	defer miner.wg.Done()

	for {
		select {
		case <-miner.startCh:
			miner.worker.start()
		case <-miner.stopCh:
			miner.worker.stop()
		case <-miner.exitCh:
			miner.worker.close()
			return
		}
	}
}

func (miner *Miner) Stop() {
	miner.stopCh <- struct{}{}
}

func (miner *Miner) Close() {
	log.Debug("closing miner of this shard..", "shardID", miner.chain.GetChainID())
	close(miner.exitCh)
	miner.wg.Wait()
	log.Debug("miner of this shard has been close!", "shardID", miner.chain.GetChainID())
}
