package data

/**
 * 交易分配生成：map[txid] sharid
 */

import (
	"encoding/csv"
	"go-w3chain/client"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"go-w3chain/shard"
	"io"
	"math/big"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

var (
	alltxs []*core.Transaction

	txtable   map[uint64]int         // txid 映射到 shardID
	addrTable map[common.Address]int // 账户地址 映射到 shardID

	tx2ClientTable map[uint64]int // txid 映射到客户端ID

)

/**
 * 加载数据集, maxNum=-1则全加载
 */
func LoadETHData(filepath string, maxTxNum int) {
	f, err := os.Open(filepath)
	if err != nil {
		log.Error("Open dataset error!", ":", err)
		return
	}
	reader := csv.NewReader(f)
	colname, err := reader.Read()
	// log.Debug("show label ", "colname:", colname)
	if err != nil {
		log.Error("Get reader handle error!", "err:", err, "colname:", colname)
		return
	}
	txid := uint64(0)
	for {
		row, err := reader.Read()
		if err != nil && err != io.EOF {
			log.Error("Read transaction error!", ":", err)
			return
		}
		if err == io.EOF {
			break
		}
		// 该字段为 input，若不为空则表示调用或创建了智能合约，这类交易应该排除掉
		// if row[6] != "0x" {
		// 	continue
		// }
		sender := common.HexToAddress(row[0][2:])
		recipient := common.HexToAddress(row[1][2:])
		value := new(big.Int)
		// value.SetString(row[6], 10)
		value.SetString("1", 10)
		tx := core.Transaction{
			TXtype:    core.UndefinedTXType,
			Sender:    &sender,
			Recipient: &recipient,
			Value:     value,
			ID:        txid,
		}
		alltxs = append(alltxs, &tx)
		txid += 1
		if (maxTxNum > 0) && (txid >= uint64(maxTxNum)) {
			break
		}
	}
	log.Debug("Load data completed", "total number", len(alltxs), "first tx", alltxs[0])
	result.SetTotalTXNum(len(alltxs))
}

/**
* 实现交易到客户端的划分
 */
func SetTX2ClientTable(clientNum int) {
	tx2ClientTable = make(map[uint64]int)
	for _, tx := range alltxs {
		tx2ClientTable[tx.ID] = int(tx.ID) % clientNum
	}
}

/**
* 注入交易到客户端，一次性全部注入
 */
func InjectTX2Client(clients []*client.Client) {
	clientNum := len(clients)
	txlist4EachClient := make([][]*core.Transaction, clientNum)
	for i, _ := range txlist4EachClient {
		txlist4EachClient[i] = make([]*core.Transaction, 0, len(alltxs)*2/clientNum)
	}
	for _, tx := range alltxs {
		cid := tx2ClientTable[tx.ID]
		txlist4EachClient[cid] = append(txlist4EachClient[cid], tx)
	}
	for i, _ := range clients {
		// log.Debug("InjectTX2Client", "clientID", i, "txNum", len(txlist4EachClient[i]))
		clients[i].Addtxs(txlist4EachClient[i])
	}
}

/**
 * 实现账户状态分片，目前基于 id 尾数
 */
func SetAddrTable(shardNum int) {
	addrTable = make(map[common.Address]int)
	for _, tx := range alltxs {
		sid := Addr2Shard(tx.Sender.Hex(), shardNum) // id = 0,1,..
		addrTable[*tx.Sender] = sid
		to_sid := Addr2Shard(tx.Recipient.Hex(), shardNum) // id = 0,1,..
		addrTable[*tx.Recipient] = to_sid
	}
}

func GetAddrTable() map[common.Address]int {
	return addrTable
}

/**
 * 根据 尾数 id 划分
 */
func Addr2Shard(addr string, shardNum int) int {
	// 只取地址后五位已绝对够用
	addr = addr[len(addr)-5:]
	num, err := strconv.ParseInt(addr, 16, 32)
	// num, err := strconv.ParseInt(senderAddr, 10, 32)
	if err != nil {
		log.Error("Parse address to shardID error!", "err:", err)
	}
	return int(num) % shardNum
}

/**
 * 取所有交易的sender，在其对应分片中提前设置状态（金额）
 */
func SetShardsInitialState(shards []*shard.Shard) {
	shardNum := len(shards)

	/* 各分片拥有的初始sender地址列表 */
	shardAddrs := make([]map[common.Address]struct{}, shardNum)
	for i := 0; i < shardNum; i++ {
		shardAddrs[i] = make(map[common.Address]struct{})
	}

	for _, tx := range alltxs {
		sid, exist := addrTable[*tx.Sender]
		if exist {
			shardAddrs[sid][*tx.Sender] = struct{}{}
		} else {
			log.Warn("this addr does not exist in addrTable (addr -> shard)", *tx.Sender)
		}
	}

	/* 将所有sender在对应分片中的初始金额设为一个极大值，确保之后注入的交易顺利执行 */
	maxValue := new(big.Int)
	maxValue.SetString("10000000000", 10)
	for i, shard := range shards { // id = 0,1,..
		shard.SetInitialState(shardAddrs[i], maxValue)
	}
	log.Info("Each shard setShardsInitialState successed")
}

func PrintTXs(num int) {
	if num == -1 {
		num = len(alltxs)
	}
	for i := 0; i < num; i++ {
		log.Debug("shows TXs", "tx", alltxs[i])
		// log.Trace("shows TXs", "id", alltxs[i].ID, "broadcast time", alltxs[i].Timestamp, "Confirm time", alltxs[i].ConfirmTimestamp)
	}
}
