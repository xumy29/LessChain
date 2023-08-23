package core

import (
	"crypto/ecdsa"
	"fmt"
	"go-w3chain/log"
	"go-w3chain/utils"
	"path/filepath"

	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	KeyStoreDir string = "keystore"
	/* 默认账户的名字，同时也是其钱包密码 */
	defaultAccountName string = "account1"
)

/** w3chain中的账户结构，对以太坊的keystore做了简单封装
 * 提供创建账户、对消息哈希进行签名、验证签名的功能
 * 目前一个节点对应一个账户
 */
type W3Account struct {
	privateKey  *ecdsa.PrivateKey
	pubKey      *ecdsa.PublicKey
	accountAddr common.Address
	keyDir      string
}

func NewW3Account(nodeDatadir string) *W3Account {
	_privateKey := newPrivateKey()

	w3Account := &W3Account{
		privateKey: _privateKey,
		keyDir:     filepath.Join(nodeDatadir, KeyStoreDir),
	}
	w3Account.pubKey = &w3Account.privateKey.PublicKey
	w3Account.accountAddr = crypto.PubkeyToAddress(*w3Account.pubKey)
	// fmt.Printf("create addr: %v\n", w3Account.accountAddr)
	return w3Account
}

func newPrivateKey() *ecdsa.PrivateKey {
	// 选择椭圆曲线，这里选择 secp256k1 曲线
	s, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		log.Error("generate private key fail", "err", err)
	}
	privateKey := s.ToECDSA()
	return privateKey
}

func (w3Account *W3Account) SignHash(hash []byte) []byte {
	sig, err := crypto.Sign(hash, w3Account.privateKey)
	if err != nil {
		log.Error("signHashFail", "err", err)
		return []byte{}
	}
	// log.Trace("w3account sign hash", "msgHash", hash, "address", w3Account.accountAddr, "sig", sig)

	return sig
}

/* 这个方法是被该账户以外的其他账户调用，以验证签名的正确性的
所以不能直接获取公钥和地址，要从签名中恢复
*/
func VerifySignature(msgHash []byte, sig []byte, expected_addr common.Address) bool {
	// 恢复公钥
	pubKeyBytes, err := crypto.Ecrecover(msgHash, sig)
	if err != nil {
		log.Error("ecrecover fail", "err", err)
		// fmt.Printf("ecrecover err: %v\n", err)
	}

	pubkey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		log.Error("UnmarshalPubkey fail", "err", err)
		// fmt.Printf("UnmarshalPubkey err: %v\n", err)
	}

	recovered_addr := crypto.PubkeyToAddress(*pubkey)
	return recovered_addr == expected_addr
}

/* 接收一个随机种子，用私钥生成一个随机数输出和对应的证明 */
func (w3Account *W3Account) GenerateVRFOutput(randSeed []byte) *utils.VRFResult {
	// vrfResult := utils.GenerateVRF(w3Account.privateKey, randSeed)
	sig := w3Account.SignHash(randSeed)
	vrfResult := &utils.VRFResult{
		RandomValue: sig,
		Proof:       w3Account.accountAddr[:],
	}
	return vrfResult
}

/* 接收随机数输出和对应证明，用公钥验证该随机数输出是否合法 */
func (w3Account *W3Account) VerifyVRFOutput(vrfResult *utils.VRFResult, randSeed []byte) bool {
	// return utils.VerifyVRF(w3Account.pubKey, randSeed, vrfResult)
	return VerifySignature(randSeed, vrfResult.RandomValue, common.BytesToAddress(vrfResult.Proof))
}

func printAccounts(w3Account *W3Account) {
	log.Info("node account", "keyDir", w3Account.keyDir, "address", w3Account.accountAddr)
	log.Info(fmt.Sprintf("privateKey: %x", crypto.FromECDSA(w3Account.privateKey)))
}

func (w3Account *W3Account) GetAccountAddress() *common.Address {
	return &w3Account.accountAddr
}

// ganache 私链上有钱的账户私钥，用来发起提交信标的交易
// 目前每个账户负责一个分片的交易
var GanacheChainAccounts []string = []string{
	"d12c928f281ed4a05f6dbf434bbbac5706826d9b2e3966077ef580df14073eb3",
	"635ccd7f8cb78b293486ee535a8aac38d33b400e4833ed07d39d2841995e0cd6",
	"831d55e90f4a55085ccf8a9acf849d9a6ce00f46fb430e47118d23af308e1486",
	"d2c42ed9778acdf7f86ce013f5437cfa463f417c0523e5ceaa9e1f8ed48eef5e",
	"26ea2b1eebb43a50c0fc5f5451073ec0831f85621765fabad93a61132cb14d21",
	"1d41896df03f6971785b1e3927daa4eed3df9113a267d953b10bfd34775a1ef4",
	"127ab599973981d4282221e339386b34c15a6b1685e0062ce388afb2ac3f1610",
	"0406fd4b37b0fef67a4cd1ca447452a0fbe81ec972e8437c2d278614295d2412",
}

// geth 私链上有钱的账户私钥，用来发起提交信标的交易
// 目前每个账户负责一个委员会的交易
var GethChainAccounts []string = []string{
	"a65d8aa17661de2eebf80c481c1d359558c3674fdc6ad916ff56e468710f5fb9",
	"4454d10f67d9470646044fd8bad1f0fc6f1ba7f20045695389b38844c0e1f835",
	"f2f0a5d8ff4d1ee1cbda99e6313063f0977357db544ad2c05787071d5ee8a044",
	"fb235001be3bc0e2500e74b66ef11004c62ed57b15d07e2766a04e0034eb261c",
	"8cd8a4bd900eb2628f52429983f9f25342358b23fa4ed99d7d343c41badcf16a",
	"f8108cd35deae7352e45ec693cdea8e38e3bd85bbbaf85332e5287af228b836d",
	"c9856af76f135a475350b5727dad741352f0418c4a7f521c3e8cc9f5adf2a074",
	"ce0f724e265d4814dd65e7d70287915d3b10915333db7b6787cd35ed5c3b1b19",
	"077c0fd242868369ca2767c340a7b7c0614266aa3ac046b5bae9b86302c50737",
}

/* 下面两个函数是不用以太坊库实现的签名和验证方法 */
// func (w3Account *W3Account) SignHash(hash []byte) []byte {
// 	r, s, err := ecdsa.Sign(rand.Reader, w3Account.privateKey, hash)
// 	if err != nil {
// 		log.Error("signHashFail", "err", err)
// 		return []byte{}
// 	}
// 	signature := append(r.Bytes(), s.Bytes()...)
// 	log.Trace("w3account sign hash", "msgHash", hash, "address", w3Account.accountAddr, "sig", signature)

// 	return signature
// }

// func (w3Account *W3Account) VerifySignature(msgHash []byte, sig []byte) bool {
// 	rBytes := sig[:len(sig)/2]
// 	sBytes := sig[len(sig)/2:]
// 	r := new(big.Int).SetBytes(rBytes)
// 	s := new(big.Int).SetBytes(sBytes)

// 	valid := ecdsa.Verify(w3Account.pubKey, msgHash, r, s)

// 	return valid
// }
