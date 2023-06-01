package beaconChain

import (
	"go-w3chain/core"
	"go-w3chain/log"
	"time"
)

type TBBlock struct {
	/** key是分片ID，value是该分片未在信标链上链的区块的信标
	 * 一个分片可能对应多个信标，取决于分片出块速度 */
	Tbs map[int][]*ConfirmedTB
	/* 时间戳 */
	Time   uint64
	Height uint64
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

	now := time.Now().Unix()
	tbChain.height += 1

	confirmTBs := make(map[int][]*ConfirmedTB, 0)
	for shardID, tbs := range tbChain.tbs_new {
		shardContract := tbChain.contract.contracts[shardID]
		for _, signedTb := range tbs {
			// todo: 验证多签名
			if !shardContract.VerifyTimeBeacon(signedTb) {
				log.Error("TBchain verify time beacon fail. this time beacon has no enough valid signatures!!")
			} else {
				log.Trace("TBchain verify time beacon success.")
			}
			confirmedTB := &ConfirmedTB{
				TimeBeacon:    signedTb.TimeBeacon,
				ConfirmTime:   uint64(now),
				ConfirmHeight: tbChain.height,
			}
			confirmTBs[shardID] = append(confirmTBs[shardID], confirmedTB)
		}
		tbChain.tbs[shardID] = append(tbChain.tbs[shardID], confirmTBs[shardID]...)
	}

	block := &TBBlock{
		Tbs:    confirmTBs,
		Time:   uint64(now),
		Height: tbChain.height,
	}

	tbChain.tbs_new = make(map[int][]*SignedTB)

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
	tbChain.PushBlock2Coms(block)
}

/** 信标链生成新区块后，将区块（包含新的信标）发送给客户端
 */
func (tbChain *BeaconChain) PushBlock2Clients(block *TBBlock) {
	tbChain.messageHub.Send(core.MsgTypeTBChainPushTB2Clients, 0, block, nil)
}

/** 信标链生成新区块后，将区块（包含新的信标）发送给委员会
 */
func (tbChain *BeaconChain) PushBlock2Coms(block *TBBlock) {
	tbChain.messageHub.Send(core.MsgTypeTBChainPushTB2Coms, 0, block, nil)
}
