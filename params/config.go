package params

import(
	"math/big"
)

type ChainConfig struct {
	ChainID *big.Int `json:"chainId"` // chainId identifies the current chain and is used for replay protection

}

var (
	Shard1Config = &ChainConfig{
		ChainID:             big.NewInt(1),
	}


	AllProtocolChanges = &ChainConfig{
		ChainID:             big.NewInt(-1),
	}
)
