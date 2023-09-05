package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

type ComGetState struct {
	From_comID     uint32
	Target_shardID uint32
	AddrList       []common.Address
}

type ShardSendState struct {
	StateDB *state.StateDB
	Height  *big.Int
}

type Msg struct {
	MsgType string
	Data    []byte
}
