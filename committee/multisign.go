package committee

import (
	"encoding/hex"
	"go-w3chain/beaconChain"
	"go-w3chain/core"
	"go-w3chain/utils"

	"github.com/ethereum/go-ethereum/common"
)

/** 模拟委员会中的节点对信标进行多签名
 * 为了方便，由委员会直接控制所有节点，从中筛选出验证者，并调用节点的方案进行签名
 */
func (com *Committee) multiSign(tb *beaconChain.TimeBeacon) *beaconChain.SignedTB {
	// 1. 获取验证者列表
	validators := com.GetValidators(com.tbchain_block_hash)

	// 2. 验证者分别对信标哈希进行签名
	sigs := make([][]byte, len(validators))
	signers := make([]common.Address, len(validators))
	for i, acc := range validators {
		signers[i] = *acc.GetAccountAddress()
		sig := acc.SignHash(tb.Hash())
		sigs[i] = sig
	}

	return &beaconChain.SignedTB{
		TimeBeacon: *tb,
		Signers:    signers,
		Sigs:       sigs,
	}
}

/** 根据最新确认的信标链区块哈希决定对信标进行多签名的验证者
 */
func (com *Committee) GetValidators(hash common.Hash) []*core.W3Account {
	// 所有节点通过vrf生成随机数
	vrfResults := make([]*utils.VRFResult, len(com.Nodes))
	for i := 0; i < len(com.Nodes); i++ {
		vrfResult := com.Nodes[i].GetAccount().GenerateVRFOutput(hash[:])
		vrfResults[i] = vrfResult
	}

	// 验证vrf结果，选出符合条件的vrf对应的账户
	validator_num := com.config.MultiSignRequiredNum
	validators := make([]*core.W3Account, 0)
	for i := 0; i < len(com.Nodes); i++ {
		valid := com.Nodes[i].GetAccount().VerifyVRFOutput(vrfResults[i], hash[:])
		if !valid {
			continue
		}
		if vrfResultIsGood(vrfResults[i].RandomValue) {
			validators = append(validators, com.Nodes[i].GetAccount())
			if len(validators) >= validator_num {
				break
			}
		}
	}

	return validators
}

/* 判断一个VRF生成的随机数是否满足条件 */
func vrfResultIsGood(val []byte) bool {
	hex := hex.EncodeToString(val)
	return hex[len(hex)-1] != '0'
}
