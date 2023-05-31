package core

import (
	"go-w3chain/log"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
)

const (
	KeyStoreDir string = "keystore" // Path within the datadir to the keystore
)

/** w3chain中的账户结构，对以太坊的keystore做了简单封装
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
	names := []string{"account1"}
	createAccounts(w3Account.ks, names)
	printAccounts(w3Account.ks)

	return w3Account
}

// func (w3Account *W3Account) SignHash(hash []byte) []byte {
// 	ks := w3Account.ks
// 	signature, err := ks.SignHash(ks.Accounts()[0], hash)
// 	if err != nil {
// 		log.Error("signHashFail", "err", err)
// 		return []byte{}
// 	}

// 	return signature
// }

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
	// walls := ks.Wallets()

	// fmt.Printf("keystore Database has %d Wallets:\n", len(walls))

	// for _, wall := range walls {
	// 	fmt.Println(wall.Accounts())
	// }

	for _, acc := range ks.Accounts() {
		log.Info("new account", "info", acc)
	}

}

// func (w3account *W3Account) CreateAccount() {

// }

// // deault password is the name
// // cmd/geth/accountcmd.go
// func accountCreate(password string) (accounts.Account, error) {
// 	scryptN := keystore.StandardScryptN
// 	scryptP := keystore.StandardScryptP
// 	keydir := KeyStoreDir

// 	// go-ethereum/accounts/keystore:passphrase.go
// 	account, err := keystore.StoreKey(keydir, password, scryptN, scryptP)
// 	if err != nil {
// 		utils.Fatalf("Failed to create account: %v", err)
// 		os.Exit(1)
// 	}
// 	fmt.Printf("\nYour new key was generated\n\n")
// 	fmt.Printf("Public address of the key:   %s\n", account.Address.Hex())
// 	fmt.Printf("Path of the secret key file: %s\n\n", account.URL.Path)
// 	fmt.Printf("- You can share your public address with anyone. Others need it to interact with you.\n")
// 	fmt.Printf("- You must NEVER share the secret key with anyone! The key controls access to your funds!\n")
// 	fmt.Printf("- You must BACKUP your key file! Without the key, it's impossible to access account funds!\n")
// 	fmt.Printf("- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!\n\n")
// 	return account, nil
// }

// func ClearKeyStore() {
// 	os.RemoveAll(Defaultkeydir)
// }

// func createGenesisAccountAndClear() {
// 	ks := createKeyStore()

// 	var name []string
// 	name = append(name, "alice")
// 	createAccountsKeyStore(ks, name)

// 	printAccountsKeyStore(ks)

// 	ClearKeyStore()
// }
