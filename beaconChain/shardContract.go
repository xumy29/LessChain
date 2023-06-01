package beaconChain

import (
	"go-w3chain/log"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"golang.org/x/crypto/sha3"
)

/** 模拟信标链上的一个合约，验证对应分片的信标的多签名
 */
type ShardContract struct {
	shardID int
	/* 一笔调用该合约的交易需要多少位验证者共同签名才会有效 */
	required_validators_num_for_sign int
}

func NewShardContract(shardID int, required int) *ShardContract {
	contract := &ShardContract{
		shardID:                          shardID,
		required_validators_num_for_sign: required,
	}

	return contract
}

/** 验证一个地址是否有资格对该信标进行签名
 * 地址长度：20个字节
 * 未完成
 */
func checkAddressValidity(address []byte) bool {
	// todo: 验证该地址
	return true
}

/** 由完整公钥推出地址
 * 公钥长度：65个字节
 * 地址长度：20个字节
 */
func pubkey2Addr(pubkey []byte) []byte {
	// 由公钥通过keccak256哈希算法得到32字节的压缩公钥
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(pubkey[1:])
	pubkeyc := hasher.Sum(nil)

	// 压缩公钥后20个字节即为账户的地址
	accountAddress := pubkeyc[len(pubkeyc)-20:]
	return accountAddress
}

func (contract *ShardContract) VerifyTimeBeacon(tb *SignedTB) bool {
	msgHash := tb.TimeBeacon.Hash().Bytes()
	sig_num := 0
	for _, sig := range tb.Sigs {
		pubkey, err := secp256k1.RecoverPubkey(msgHash, sig)
		if err != nil {
			log.Error("shardContract recover Pubkey Fail.", "err", err)
			continue
		}
		if !checkAddressValidity(pubkey2Addr(pubkey)) {
			log.Error("shardContract check address validity fail. This address has no right to sign this time beacon.")
			continue
		}
		sig = sig[:len(sig)-1] // remove recovery id
		checkSigPass := secp256k1.VerifySignature(pubkey, msgHash, sig)
		if checkSigPass {
			sig_num += 1
			if sig_num >= contract.required_validators_num_for_sign {
				return true
			}
		}
	}

	return false
}
