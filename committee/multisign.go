package committee

import (
	"go-w3chain/beaconChain"
	"go-w3chain/core"
	"go-w3chain/utils"

	"github.com/ethereum/go-ethereum/common"
)

/** 模拟委员会中的节点对信标进行多签名
 * 为了方便，由委员会直接控制所有节点，从中筛选出验证者，并调用节点的方案进行签名
 */
func (com *Committee) multiSign(tb *beaconChain.TimeBeacon) *beaconChain.SignedTB {
	// 0. 获取信标链最新区块哈希和高度，哈希作为vrf的随机种子
	seed, height := beaconChain.GetEthChainLatestBlockHash(tb.ShardID)
	// 1. 获取验证者列表和对应的vrf证明
	validators, vrfResults := com.GetValidators(seed)
	vrfs := make([][]byte, len(vrfResults))

	// 2. 验证者分别对信标哈希进行签名
	sigs := make([][]byte, len(validators))
	signers := make([]common.Address, len(validators))
	for i, acc := range validators {
		signers[i] = *acc.GetAccountAddress()
		sig := acc.SignHash(tb.Hash())
		sigs[i] = sig
		vrfs[i] = vrfResults[i].RandomValue
	}

	return &beaconChain.SignedTB{
		TimeBeacon: *tb,
		Signers:    signers,
		Sigs:       sigs,
		Vrfs:       vrfs,
		SeedHeight: height,
	}
}

/** 根据最新确认的信标链区块哈希决定对信标进行多签名的验证者
 */
func (com *Committee) GetValidators(hash common.Hash) ([]*core.W3Account, []*utils.VRFResult) {
	// 所有节点通过vrf生成随机数
	vrfResults := make([]*utils.VRFResult, len(com.Nodes))
	for i := 0; i < len(com.Nodes); i++ {
		vrfResult := com.Nodes[i].GetAccount().GenerateVRFOutput(hash[:])
		vrfResults[i] = vrfResult
	}

	// 验证vrf结果，选出符合条件的vrf对应的账户
	validator_num := com.config.MultiSignRequiredNum
	validators := make([]*core.W3Account, 0)
	vrfs := make([]*utils.VRFResult, 0)
	for i := 0; i < len(com.Nodes); i++ {
		valid := com.Nodes[i].GetAccount().VerifyVRFOutput(vrfResults[i], hash[:])
		if !valid {
			continue
		}
		if vrfResultIsGood(vrfResults[i].RandomValue) {
			validators = append(validators, com.Nodes[i].GetAccount())
			vrfs = append(vrfs, vrfResults[i])
			if len(validators) >= validator_num {
				break
			}
		}
	}

	return validators, vrfs
}

/* 判断一个VRF生成的随机数是否满足条件
该方法需与合约的验证方法一致
*/
func vrfResultIsGood(val []byte) bool {
	// 注意，如果是直接用签名方式生成vrf，则最后一个字节只会是0或1
	return val[0] > 50
}
