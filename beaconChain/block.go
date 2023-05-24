package beaconChain

import (
	"go-w3chain/core"
	"go-w3chain/log"
	"time"
)

type TBBlock struct {
	/** key是分片ID，value是该分片未在信标链上链的区块的信标
	 * 一个分片可能对应多个信标，取决于分片出块速度 */
	tbs map[int][]*TimeBeacon
}

func (tbChain *BeaconChain) loop() {
	defer tbChain.wg.Done()
	log.Info("TBChain work loop start.")
	blockInterval := time.Duration(tbChain.blockIntervalSecs) * time.Second
	timer := time.NewTimer(blockInterval)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			block := tbChain.GenerateBlock()
			tbChain.PushBlock(block)
			timer.Reset(blockInterval)
		case <-tbChain.stopCh:
			log.Info("TBChain work loop stop.")
			return
		}
	}
}

func (tbChain *BeaconChain) GenerateBlock() *TBBlock {
	tbChain.lock_new.Lock()
	defer tbChain.lock_new.Unlock()
	tbChain.lock.Lock()
	defer tbChain.lock.Unlock()

	for shardID, tbs := range tbChain.tbs_new {
		tbChain.tbs[shardID] = append(tbChain.tbs[shardID], tbs...)
	}

	block := &TBBlock{
		tbs: tbChain.tbs_new,
	}
	tbChain.tbs_new = make(map[int][]*TimeBeacon)

	log.Debug("tbchain generate block", "info", block)
	return block
}

/** 信标链生成新区块后，将区块（包含新的信标）发送给订阅者
* 实际情况下，应该是有监督节点监听信标链的新区块，并将其中的信标发送给订阅者。
这里进行了简化，直接跳过监督节点，由信标链发送给订阅者
* 订阅者包括客户端、委员会等需要获取信标辅助验证的角色
*/
func (tbChain *BeaconChain) PushBlock(block *TBBlock) {
	tbChain.PushBlock2Clients(block)
}

/** 信标链生成新区块后，将区块（包含新的信标）发送给客户端
 */
func (tbChain *BeaconChain) PushBlock2Clients(block *TBBlock) {
	tbChain.messageHub.Send(core.MsgTypeTBChainPushTB2Clients, 0, block.tbs, nil)
}
