package ganache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
)

var (
	client       *ethclient.Client
	contractAddr common.Address
	contractABI  *abi.ABI
)

func TestConnect(t *testing.T) {
	var err error
	client, err = Connect(7545)
	assert.Equal(t, true, err == nil)

	id, err := client.NetworkID(context.Background())
	assert.Equal(t, true, err == nil)
	assert.Equal(t, int64(5777), id.Int64())
}

func TestDeploy(t *testing.T) {
	var err error
	genesisTBs := make([]ContractTB, 2)
	genesisTBs[0] = ContractTB{
		Height:  0,
		ShardID: 0,
	}
	genesisTBs[1] = ContractTB{
		Height:    0,
		ShardID:   1,
		BlockHash: "0x11111",
	}

	contractAddr, contractABI, _, err = DeployContract(client, genesisTBs)
	// fmt.Println(contractAddr)
	// fmt.Printf("%v\n", abi)
	assert.Equal(t, true, err == nil)

	eventChannel := make(chan *Event, 10)
	go SubscribeEvents(7545, contractAddr, eventChannel)
}

func TestUseContract(t *testing.T) {
	// got, err := GetTB(client, contractAddr, contractABI, 1, 0)
	// assert.Equal(t, true, err == nil)
	// expected := &ContractTB{
	// 	Height:    0,
	// 	ShardID:   1,
	// 	BlockHash: "0x11111",
	// }
	// assert.Equal(t, expected, got)

	// newTB := &ContractTB{
	// 	Height:     1,
	// 	ShardID:    0,
	// 	StatusHash: "0x123456",
	// }
	// err = AddTB(client, contractAddr, contractABI, newTB)
	// assert.Equal(t, true, err == nil)

	// got, err = GetTB(client, contractAddr, contractABI, 0, 1)
	// assert.Equal(t, true, err == nil)
	// assert.Equal(t, newTB, got)

	for i := 0; i < 4; i++ {
		time.Sleep(5 * time.Second)
		newTB := &ContractTB{
			Height:     uint64(i + 2),
			ShardID:    0,
			StatusHash: fmt.Sprintf("0xaaaa%d", i),
		}
		err := AddTB(client, contractAddr, contractABI, newTB)
		assert.Equal(t, true, err == nil)
	}
}
