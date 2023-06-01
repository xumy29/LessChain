package core

import (
	"fmt"
	"go-w3chain/log"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
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
	names := []string{defaultAccountName}
	createAccounts(w3Account.ks, names)

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
		log.Info("new account", "info", acc)
	}
}

func (w3Account *W3Account) SignHash(hash []byte) []byte {
	ks := w3Account.ks
	signature, err := ks.SignHashWithPassphrase(ks.Accounts()[0], defaultAccountName, hash)
	if err != nil {
		// log.Error("signHashFail", "err", err)
		fmt.Print("signHashFail", "err", err)
		return []byte{}
	}

	return signature
}

func (w3Account *W3Account) VerifySignature(msgHash []byte, sig []byte) bool {
	pubkey, err := secp256k1.RecoverPubkey(msgHash, sig)
	if err != nil {
		// log.Error("recover Pubkey Fail.")
		fmt.Print("recover Pubkey Fail.")
		return false
	}
	sig = sig[:len(sig)-1] // remove recovery id
	return secp256k1.VerifySignature(pubkey, msgHash, sig)
}
