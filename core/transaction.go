package core

import (
	"bytes"
	"go-w3chain/result"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// 在变量后添加 json:"xxxxx"字符串 并且注意字符串使用``包裹
/*
Type: 0：片内，  1：CTX1,	2:CTX2
*/

const (
	UndefinedTXType uint64 = iota
	IntraTXType
	CrossTXType1 // 跨片交易前半部分
	CrossTXType2 // 跨片交易后半部分
	RollbackTXType
)

type Transaction struct {
	TXtype uint64 `json:"type"`
	ID     uint64

	/* 普通交易相关 */
	Sender      *common.Address `json:"sender"`
	Recipient   *common.Address `json:"recipient"`
	SenderNonce uint64          `json:"senderNonce"`
	Value       *big.Int        `json:"value"`

	/** 记录timestamp(求tps, latency)， 账户的分片id(求跨分片比例，负载)
	 * 注：int类型不能rlp
	 */
	Timestamp        uint64 // broadcast Timestamp
	ConfirmTimestamp uint64
	Sender_sid       uint64
	Recipient_sid    uint64
	TXStatus         uint64

	/* 客户端信息 */
	Cid          uint64
	RollbackSecs uint64

	// // Signature values of sender
	// V1 *big.Int `json:"v1"`
	// R1 *big.Int `json:"r1"`
	// S1 *big.Int `json:"s1"`

	// // Signature values of broker
	// V2 *big.Int `json:"v2"`
	// R2 *big.Int `json:"r2"`
	// S2 *big.Int `json:"s2"`

	// Payload      []byte          `json:"input"`
}

type Transactions []*Transaction
type DerivableList interface {
	Len() int
	EncodeIndex(int, *bytes.Buffer)
}

// Len returns the length of s.
func (s Transactions) Len() int { return len(s) }

// EncodeIndex encodes the i'th transaction to w. Note that this does not check for errors
// because we assume that *Transaction will only ever contain valid txs that were either
// constructed by decoding or via public API in this package.
func (s Transactions) EncodeIndex(i int, w *bytes.Buffer) {
	tx := s[i]
	rlp.Encode(w, tx)
}

func (t Transaction) GetRLP() ([]byte, error) {
	return rlp.EncodeToBytes(t)
}

//
func NewTransaction(txtype uint64, sender, to common.Address, senderNonce uint64, amount *big.Int) *Transaction {
	tx := Transaction{
		TXtype: txtype,

		Sender:    &sender,
		Recipient: &to,

		SenderNonce: senderNonce,

		Value: amount,
	}
	return &tx

}

/* 深拷贝， status 设置为 DefaultStatus */
func CopyTransaction(oldtx *Transaction) *Transaction {
	tx := Transaction{
		TXtype: oldtx.TXtype,
		ID:     oldtx.ID,

		Sender:      oldtx.Sender,
		Recipient:   oldtx.Recipient,
		SenderNonce: oldtx.SenderNonce,
		Value:       oldtx.Value,

		Timestamp:        oldtx.Timestamp,
		ConfirmTimestamp: oldtx.ConfirmTimestamp,
		Sender_sid:       oldtx.Sender_sid,
		Recipient_sid:    oldtx.Recipient_sid,
		TXStatus:         result.DefaultStatus,
	}
	return &tx
}

// Nonce returns the specific account nonce of the transaction.
func (tx *Transaction) Nonce() uint64 {
	return tx.SenderNonce
}

// NewCoinbaseTX creates a new coinbase transaction
// func NewCoinbaseTX(to, data string) *Transaction {
// 	tx := Transaction{}
// 	return &tx
// }

// TxByNonce implements the sort interface to allow sorting a list of transactions
// by their nonces. This is usually only useful for sorting transactions from a
// single account, otherwise a nonce comparison doesn't make much sense.
type TxByNonce Transactions

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].Nonce() < s[j].Nonce() }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
