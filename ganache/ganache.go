package ganache

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"go-w3chain/core"
	"go-w3chain/log"
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

var (
	chainID int
)

func SetChainID(id int) {
	chainID = id
}

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
	account := core.GanachePublicAccount
	if shardID >= 0 {
		account = core.GanacheChainAccounts[shardID]
	}
	privateKey, err := crypto.HexToECDSA(account)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// 部署合约
func DeployContract(client *ethclient.Client,
	genesisTBs []ContractTB,
	required_sig_cnt uint32) (common.Address, *abi.ABI, *big.Int, error) {

	// 编译 Solidity 合约并获取合约 ABI 和字节码
	contractABI, err := abi.JSON(strings.NewReader(myContractABI()))
	if err != nil {
		return common.Address{}, nil, big.NewInt(0), err
	}
	bytecode := common.FromHex(myContractByteCode())

	// 获取私钥
	privateKey, err := myPrivateKey(-1)
	if err != nil {
		return common.Address{}, nil, big.NewInt(0), err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(int64(chainID)))
	if err != nil {
		fmt.Println("bind.NewKeyedTransactorWithChainID err: ", err)
		return common.Address{}, nil, big.NewInt(0), err
	}

	// 部署合约
	address, tx, _, err := bind.DeployContract(auth, contractABI, bytecode, client, genesisTBs, required_sig_cnt)
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
	lastNonce map[uint32]uint64 = make(map[uint32]uint64)
	// 通过该锁使不同委员会的AddTB方法串行执行，避免一些并发调用导致的问题
	call_lock      sync.Mutex
	lowestGasPrice *big.Int = big.NewInt(0)
)

// 存储信标到合约
func AddTB(client *ethclient.Client, contractAddr common.Address,
	abi *abi.ABI, tb *ContractTB, sigs [][]byte, signers []common.Address) error {

	call_lock.Lock()
	defer call_lock.Unlock()

	var tmpShardID uint32 = 0
	// tmpShardID = tb.ShardID

	// 构造调用数据
	callData, err := abi.Pack("addTB", *tb, sigs, signers)
	if err != nil {
		fmt.Println("abi.Pack err: ", err)
		return err
	}

	// 通过私钥构造签名者
	privateKey, err := myPrivateKey(int(tmpShardID))
	if err != nil {
		fmt.Println("get myPrivateKey err: ", err)
		return err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(int64(chainID)))
	if err != nil {
		fmt.Println("bind.NewKeyedTransactorWithChainID err: ", err)
		return err
	}

	// 设置交易参数
	auth.GasLimit = uint64(30000000) // 设置 gas 限制
	auth.Value = big.NewInt(0)       // 设置发送的以太币数量（如果有的话）

	var nonce uint64
	_, ok := lastNonce[tmpShardID]
	if !ok {
		// 如果在之前的交易中使用了相同的账户地址，而这些交易还未被确认（被区块打包），那么下一笔交易的nonce应该是
		// 当前账户的最新nonce+1。
		nonce, err = client.PendingNonceAt(context.Background(), auth.From)
		if err != nil {
			log.Error("client.PendingNonceAt err", "err", err)
			fmt.Println("client.PendingNonceAt err: ", err)
			return err
		}

		lastNonce[tmpShardID] = nonce
	} else {
		nonce = lastNonce[tmpShardID] + 1
		lastNonce[tmpShardID] = nonce
	}
	// nonce_lock.Unlock()

	if lowestGasPrice.Uint64() == 0 {
		gasPrice, err := client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Error("client.SuggestGasPrice err", "err", err)
			fmt.Println("client.SuggestGasPrice err: ", err)
			return err
		}
		lowestGasPrice = gasPrice
	}

	// mustSend：循环发送直到交易发送成功
	for true {
		tx := types.NewTransaction(nonce, contractAddr, big.NewInt(0), auth.GasLimit, lowestGasPrice, callData)

		// 签名交易
		signedTx, err := auth.Signer(auth.From, tx)
		if err != nil {
			log.Error("auth.Signer err", "err", err)
			fmt.Println("auth.Signer err: ", err)
			return err
		}

		// 发送交易
		err = client.SendTransaction(context.Background(), signedTx)
		if err != nil {
			log.Debug("client.SendTransaction err", "err", err, "txtype", "AddTB", "shardID", tb.ShardID, "height", tb.Height,
				"gasPrice", lowestGasPrice, "nonce", nonce)
			fmt.Println("client.SendTransaction err: ", err)

			if err.Error() != "transaction underpriced" {
				return err
			} else {
				// 每次提升10%的手续费，并将该手续费作为最低手续费
				toAdd := new(big.Int).Div(lowestGasPrice, big.NewInt(10))
				lowestGasPrice = new(big.Int).Add(lowestGasPrice, toAdd)
			}
		} else {
			// fmt.Printf("signedTX: %v\n", signedTx.Hash().Hex())
			break
		}
	}

	// log.Debug("sendTransaction success!", "txtype", "AddTB", "shardID", tb.ShardID, "height", tb.Height,
	// 	"gasPrice", lowestGasPrice, "nonce", nonce)

	// pendingCnt, err := client.PendingTransactionCount(context.Background())
	// if err != nil {
	// 	log.Debug("client.PendingTransactionCount err", "err", err)
	// } else {
	// 	log.Debug("client.PendingTransactionCount", "cnt", pendingCnt)
	// }
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
