package committee

import (
	"go-w3chain/beaconChain"
	"go-w3chain/core"

	"github.com/ethereum/go-ethereum/common"
)

/** 模拟委员会中的节点对信标进行多签名
 * 为了方便，由委员会直接控制所有节点，从中筛选出验证者，并调用节点的方案进行签名
 */
func (com *Committee) multiSign(tb *beaconChain.TimeBeacon) *beaconChain.SignedTB {
	// 1. 获取验证者列表
	validators := com.GetValidators(tb.BlockHash)

	// 2. 验证者分别对信标哈希进行签名
	sigs := make([][]byte, len(validators))
	for i, acc := range validators {
		sig := acc.SignHash(tb.Hash().Bytes())
		sigs[i] = sig
	}

	return &beaconChain.SignedTB{
		TimeBeacon: tb,
		Sigs:       sigs,
	}
}

/** 根据最新区块哈希决定对信标进行多签名的验证者地址
 * 这里暂时不使用VRF方法，直接取排在前面的节点，节省时间
 */
func (com *Committee) GetValidators(hash common.Hash) []*core.W3Account {
	validator_num := com.config.MultiSignRequiredNum
	validators := make([]*core.W3Account, validator_num)
	for i := 0; i < validator_num; i++ {
		validators[i] = com.Nodes[i].GetAccount()
	}
	return validators
}
