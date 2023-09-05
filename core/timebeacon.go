package core

import "github.com/ethereum/go-ethereum/common"

type TimeBeacon struct {
	ShardID    uint32 `json:"shardID" gencodec:"required"`
	Height     uint64 `json:"height" gencodec:"required"`
	BlockHash  string `json:"blockHash" gencodec:"required"`
	TxHash     string `json:"txHash" gencodec:"required"`
	StatusHash string `json:"statusHash" gencodec:"required"`
}

type SignedTB struct {
	TimeBeacon
	Sigs       [][]byte
	Vrfs       [][]byte
	SeedHeight uint64
	Signers    []common.Address
}
