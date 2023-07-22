package core

import (
	"go-w3chain/log"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
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
	keyDir string
	ks     *keystore.KeyStore
}

func NewW3Account(nodeDatadir string) *W3Account {
	w3Account := &W3Account{
		keyDir: filepath.Join(nodeDatadir, KeyStoreDir),
	}
	w3Account.ks = createKeyStore(w3Account.keyDir)
	if len(w3Account.ks.Accounts()) == 0 {
		names := []string{defaultAccountName}
		createAccounts(w3Account.ks, names)
	}

	return w3Account
}

func createKeyStore(keyDir string) *keystore.KeyStore {
	ks := keystore.NewPlaintextKeyStore(keyDir)

	return ks
}

func createAccounts(ks *keystore.KeyStore, names []string) {
	// deault password is the name
	for _, name := range names {
		ks.NewAccount(name)
	}
}

func printAccounts(ks *keystore.KeyStore) {
	for _, acc := range ks.Accounts() {
		log.Info("node account", "info", acc)
	}
}

func (w3Account *W3Account) GetAccountAddress() *common.Address {
	return &w3Account.ks.Accounts()[0].Address
}

func (w3Account *W3Account) SignHash(hash []byte) []byte {
	ks := w3Account.ks
	signature, err := ks.SignHashWithPassphrase(ks.Accounts()[0], defaultAccountName, hash)
	if err != nil {
		log.Error("signHashFail", "err", err)
		// fmt.Print("signHashFail", "err", err)
		return []byte{}
	}
	log.Trace("w3account sign hash", "msgHash", hash, "address", ks.Accounts()[0], "sig", signature)

	return signature
}

func (w3Account *W3Account) VerifySignature(msgHash []byte, sig []byte) bool {
	pubkey, err := secp256k1.RecoverPubkey(msgHash, sig)
	// fmt.Print(len(pubkey)) // 恢复出来的公钥长度为65个字节，第一个字节固定是0x04
	if err != nil {
		log.Error("recover Pubkey Fail.")
		// fmt.Print("recover Pubkey Fail.")
		return false
	}

	// 由公钥通过keccak256哈希算法得到32字节的压缩公钥
	// hasher := sha3.NewLegacyKeccak256()
	// hasher.Write(pubkey[1:])
	// pubkeyc := hasher.Sum(nil)

	// fmt.Println(pubkeyc)
	// fmt.Println(w3Account.GetAccountAddress()[:])
	// 压缩公钥后20个字节即为账户的地址
	// if !bytes.Equal(pubkeyc[12:], w3Account.GetAccountAddress()[:]) {
	// 	fmt.Print("not equal")
	// 	return false
	// }

	sig = sig[:len(sig)-1] // remove recovery id
	return secp256k1.VerifySignature(pubkey, msgHash, sig)
}

// ganache 私链上有钱的账户，用来发起提交信标的交易
// 目前每个账户负责一个委员会的交易
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
