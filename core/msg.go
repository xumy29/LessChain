package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Msg struct {
	MsgType string
	Data    []byte
}

type ComGetHeight struct {
	From_comID     uint32
	Target_shardID uint32
}

type ComGetState struct {
	From_comID     uint32
	Target_shardID uint32
	AddrList       []common.Address
}

type ShardSendState struct {
	StatusTrieHash common.Hash
	AccountData    map[common.Address][]byte
	AccountsProofs map[common.Address][][]byte
	Height         *big.Int
}

type ShardSendGenesis struct {
	Addrs           []common.Address
	Gtb             *TimeBeacon
	Target_nodeAddr string
}

type BooterSendContract struct {
	Addr common.Address
}

type ComSendBlock struct {
	Transactions []*Transaction
	Header       *Header
}
