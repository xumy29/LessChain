package core

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"fmt"
	"go-w3chain/utils"
	"os"
)

var Defaultkeydir string = filepath.Join(DefaultDataDir(), datadirDefaultKeyStore)

// deault password is the name
// cmd/geth/accountcmd.go
func accountCreate(password string) (accounts.Account, error) {
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	keydir := Defaultkeydir

	// go-ethereum/accounts/keystore:passphrase.go
	account, err := keystore.StoreKey(keydir, password, scryptN, scryptP)
	if err != nil {
		utils.Fatalf("Failed to create account: %v", err)
		os.Exit(1)
	}
	fmt.Printf("\nYour new key was generated\n\n")
	fmt.Printf("Public address of the key:   %s\n", account.Address.Hex())
	fmt.Printf("Path of the secret key file: %s\n\n", account.URL.Path)
	fmt.Printf("- You can share your public address with anyone. Others need it to interact with you.\n")
	fmt.Printf("- You must NEVER share the secret key with anyone! The key controls access to your funds!\n")
	fmt.Printf("- You must BACKUP your key file! Without the key, it's impossible to access account funds!\n")
	fmt.Printf("- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!\n\n")
	return account, nil
}

// go-ethereum/accounts/keystore.go
func createKeyStore() *keystore.KeyStore {
	keydir := Defaultkeydir
	ks := keystore.NewPlaintextKeyStore(keydir)

	return ks
}

func createAccountsKeyStore(ks *keystore.KeyStore, names []string) {
	// deault password is the name
	for _, name := range names {
		ks.NewAccount(name)
	}
}

func printAccountsKeyStore(ks *keystore.KeyStore) {
	walls := ks.Wallets()

	fmt.Printf("keystore Database has %d Wallets:\n", len(walls))

	for _, wall := range walls {
		fmt.Println(wall.Accounts())
	}

}

func ClearKeyStore() {
	os.RemoveAll(Defaultkeydir)
}

func createGenesisAccountAndClear() {
	ks := createKeyStore()

	var name []string
	name = append(name, "alice")
	createAccountsKeyStore(ks, name)

	printAccountsKeyStore(ks)

	ClearKeyStore()
}
