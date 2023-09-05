package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Shard interface {
	GetShardID() uint32
	GetBlockChain() *BlockChain
	GetChainHeight() uint64

	SetInitialAccountState(map[common.Address]struct{}, *big.Int)
	AddBlock(*Block)

	SetMessageHub(MessageHub)

	Start()
	Close()
}
