package ganache

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"go-w3chain/utils"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ContractTB struct {
	ShardID    uint32 `json:"shardID" gencodec:"required"`
	Height     uint64 `json:"height" gencodec:"required"`
	BlockHash  string `json:"blockHash" gencodec:"required"`
	TxHash     string `json:"txHash" gencodec:"required"`
	StatusHash string `json:"statusHash" gencodec:"required"`
}

func Connect(port int) (*ethclient.Client, error) {
	addr := fmt.Sprintf("http://localhost:%d", port)
	client, err := ethclient.Dial(addr)
	return client, err
}

func myPrivateKey(shardID int) (*ecdsa.PrivateKey, error) {
	privateKeyHexs := [2]string{"369bb574424350c46381ed892bff1a83db77aa40c7a7f13a82f17b95879eba08", "ce3009672aec74773eb279828ead0786bd049ddd6ca4f32d58b38a85fcf91bb1"}
	privateKey, err := crypto.HexToECDSA(privateKeyHexs[shardID])
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// 部署合约
func DeployContract(client *ethclient.Client, genesisTBs []ContractTB) (common.Address, *abi.ABI, *big.Int, error) {
	// 编译 Solidity 合约并获取合约 ABI 和字节码
	contractABI, err := abi.JSON(strings.NewReader(myContractABI()))
	if err != nil {
		return common.Address{}, nil, big.NewInt(0), err
	}
	bytecode := common.FromHex(myContractByteCode())

	// 获取私钥
	privateKey, err := myPrivateKey(0)
	if err != nil {
		return common.Address{}, nil, big.NewInt(0), err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(5777))
	if err != nil {
		fmt.Println("bind.NewKeyedTransactorWithChainID err: ", err)
		return common.Address{}, nil, big.NewInt(0), err
	}

	// // 构建交易数据
	// gasLimit := uint64(3000000)
	// gasPrice, err := client.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	fmt.Println("SuggestGasPrice err: ", err)
	// 	return common.Address{}, nil, big.NewInt(0), err
	// }

	// 部署合约
	address, tx, _, err := bind.DeployContract(auth, contractABI, bytecode, client, genesisTBs)
	if err != nil {
		fmt.Println("DeployContract err: ", err)
		return common.Address{}, nil, big.NewInt(0), err
	}

	// 等待交易被挖矿确认
	_, err = bind.WaitDeployed(context.Background(), client, tx)
	if err != nil {
		fmt.Println("WaitDeployed err: ", err)
		return common.Address{}, nil, big.NewInt(0), err
	}

	fmt.Printf("contract deploy. address: %v\n", address)

	return address, &contractABI, nil, nil
}

var (
	// nonce      uint64 = 0
	lastNonce  [2]uint64 = [2]uint64{0, 0}
	nonce_lock sync.Mutex
)

// 存储信标到合约
func AddTB(client *ethclient.Client, contractAddr common.Address, abi *abi.ABI, tb *ContractTB) error {
	// 构造调用数据
	callData, err := abi.Pack("addTB", *tb)
	if err != nil {
		fmt.Println("abi.Pack err: ", err)
		return err
	}

	// 通过私钥构造签名者
	privateKey, err := myPrivateKey(int(tb.ShardID))
	if err != nil {
		fmt.Println("get myPrivateKey err: ", err)
		return err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(5777))
	if err != nil {
		fmt.Println("bind.NewKeyedTransactorWithChainID err: ", err)
		return err
	}

	// 设置交易参数
	auth.GasLimit = uint64(300000) // 设置 gas 限制
	auth.Value = big.NewInt(0)     // 设置发送的以太币数量（如果有的话）
	// 构建交易对象
	// nonce_lock.Lock()
	// if nonce == 0 {
	// 	// nonce, err = client.PendingNonceAt(context.Background(), auth.From)
	// 	// if err != nil {
	// 	// 	fmt.Println("client.PendingNonceAt err: ", err)
	// 	// 	return err
	// 	// }
	// 	nonce = 190
	// } else {
	// 	nonce = nonce + 1
	// }
	// nonce_lock.Unlock()

	// 如果在之前的交易中使用了相同的账户地址，而这些交易还未被确认（被区块打包），那么下一笔交易的nonce应该是
	// 当前账户的最新nonce+1。
	nonce_lock.Lock()
	nonce, err := client.PendingNonceAt(context.Background(), auth.From)

	if err != nil {
		fmt.Println("client.PendingNonceAt err: ", err)
		return err
	}
	// fmt.Printf("client.PendingNonceAt nonce = %v\n", nonce)

	// fmt.Printf("lastNonce = %v\n", lastNonce[tb.ShardID])

	if lastNonce[tb.ShardID] == 0 {
		lastNonce[tb.ShardID] = nonce
	} else {
		if nonce == lastNonce[tb.ShardID] {
			nonce = nonce + 1
		}
		lastNonce[tb.ShardID] = nonce
	}
	nonce_lock.Unlock()

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		fmt.Println("client.SuggestGasPrice err: ", err)
		return err
	}
	tx := types.NewTransaction(nonce, contractAddr, big.NewInt(0), auth.GasLimit, gasPrice, callData)

	// 签名交易
	signedTx, err := auth.Signer(auth.From, tx)
	if err != nil {
		fmt.Println("auth.Signer err: ", err)
		return err
	}
	// fmt.Printf("auth.Signer nonce = %v\n", signedTx.Nonce())

	// 发送交易
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		fmt.Println("client.SendTransaction err: ", err)
		return err
	}

	fmt.Printf("signedTX: %v\n", signedTx.Hash().Hex())

	return nil
}

// 从合约读取信标
func GetTB(client *ethclient.Client, contractAddr common.Address, abi *abi.ABI, shardID uint32, height uint64) (*ContractTB, error) {
	callData, err := abi.Pack("getTB", shardID, height)
	if err != nil {
		fmt.Println("abi.Pack err: ", err)
		return nil, err
	}

	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: callData,
	}

	// callContract 向合约发送一笔只读调用
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		fmt.Println("client.CallContract err: ", err)
		return nil, err
	}
	// fmt.Printf("result: %v\n", result)

	tb := resolveTB(result, abi)

	return tb, nil
}

func resolveTB(resultData []byte, abi *abi.ABI) *ContractTB {
	response, err := abi.Methods["getTB"].Outputs.Unpack(resultData)
	if err != nil {
		fmt.Println("解析返回数据失败:", err)
		return nil
	}

	contractTB := response[0]
	// fmt.Printf("%v\n", contractTB)

	fields := utils.GetFieldValues(contractTB)
	tb := &ContractTB{
		Height:     fields["Height"].(uint64),
		ShardID:    fields["ShardID"].(uint32),
		BlockHash:  fields["BlockHash"].(string),
		TxHash:     fields["TxHash"].(string),
		StatusHash: fields["StatusHash"].(string),
	}

	return tb
}
