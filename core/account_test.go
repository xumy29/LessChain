package core

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

var (
	dataDir   = "../test/"
	w3Account *W3Account
	testmsg   = hexutil.MustDecode("0xce0677bb30baa8cf067c88db9811f4333d131bf8bcf12fe7065d211dce971008")
	sig       []byte
)

func TestCreateAccount(t *testing.T) {
	w3Account = NewW3Account(dataDir)
}

func TestSignHash(t *testing.T) {
	sig = w3Account.SignHash(testmsg)
}

func TestVerifySinature(t *testing.T) {
	if !VerifySignature(testmsg, sig, *w3Account.GetAccountAddress()) {
		t.Error("wrong verify sig")
	}
}
