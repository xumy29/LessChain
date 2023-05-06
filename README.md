# 写在前面
这份代码是ICBC2023的中稿论文 "W3Chain: A Layer2 Blockchain Defeating the Scalability Trilemma" 的仿真代码。在此基础上，对其进行了一些扩展。

# 配置运行
+ **数据集**： 以太坊第14920000个区块到第14960000个区块的数据（200w+条交易），存在本地未上传。

+ **运行模式**：分为 `debug` 模式和 `run` 模式。两个模式下分别读取的配置文件为 ".cfg/debug.json" 和 ".cfg/run.json"。

+ **参数文件设置**: 以 "./cfg/run.json" 为例
```json
{
    "LogLevel": 3,
    "LogFile": "",
    "IsProgressBar": true,
    "IsLogProgress": true,
    "LogProgressInterval": 20,
    "ClientNum": 2,
    "ShardNum": 8,
    "MaxTxNum": 400000,
    "InjectSpeed": 4000,
    "RecommitInterval": 5,
    "RecommitTimes2Rollback": 5,
    "maxBlockTXSize": 4000,
    "datasetDir": "D:/project/blockChain/ethereum_data/14920000_14960000/ExternalTransactionItem.csv"
}
```


PS: RecommitInterval 即出块间隔，必须大于等于5s，否则会被强制设为5s。可在worker.go中更改此限制。


+ **编译运行**(run模式)
``` go
go build -o w3Chain
./w3Chain -m run
```
PS：windows系统要用exe后缀，即： go build -o w3Chain.exe
    
## 编译时go get 超时：
+ 安装 go.mod 的 package时，go get 超时：需要设置代理
``` go
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
```

+ **运行日志**
存在 `LogFile` 参数指定的路径下

# 交易分配
在dataprocess.go中，读取数据集中的交易，并将交易划分到不同客户端。客户端根据交易发送者地址划分交易到不同分片，按照 `inject_speed` 将交易注入到分片中。

# 交易执行
core/transaction.go 中定义了所有交易的类型，如下
``` go
const (
	UndefinedTXType uint64 = iota
	IntraTXType
	CrossTXType1 // 跨片交易前半部分
	CrossTXType2 // 跨片交易后半部分
	RollbackTXType
)
```
初始交易类型只会是 `IntraTXType` 和 `CrossTXType1`，由客户端向 Sender 地址所在分片（源分片）发送。分片执行完一笔交易后，会向发送该交易的客户端返回一个收据。对于 `CrossTXType1` 交易，客户端收到回复后会再向该交易的 Recipient 地址所在分片发送 `CrossTXType2` 交易。为了保证交易原子性，如果一笔 `CrossTXType2` 交易超过一定时间没有被执行，则客户端向源分片发送一笔回滚交易，回滚对应的 `CrossTXType1` 交易。

# 模块介绍
## log 模块
插件，实现了日志的功能，支持不同的日志级别。
使用例子：
``` go
// Sanitize recommit interval if the user-specified one is too short.
    recommit := worker.config.Recommit
    if recommit < minRecommitInterval {
        log.Warn("Sanitizing miner recommit interval", "provided", recommit, "updated", minRecommitInterval)
        recommit = minRecommitInterval
    }
```

## messageHub 模块
用于在不同角色之间传递消息。目前是单机仿真，没有采用网络传输，而是通过 messageHub 直接调用角色的对象方法进行消息写入。

多种类型的消息通过 messageHub 进行传递，共用一个接口，callback是回调函数。
``` go
/* 用于分片、委员会、客户端、信标链传送消息 */
func (hub *GoodMessageHub) Send(msgType uint64, id uint64, msg interface{}, callback func(res ...interface{})) {
	switch msgType {
	case core.MsgTypeShardReply2Client:
		client := clients_ref[id]
		receipts := msg.([]*result.TXReceipt)
		client.AddTXReceipts(receipts)

	case core.MsgTypeClientInjectTX2Shard:
		shard := shards_ref[id]
		txs := msg.([]*core.Transaction)
		shard.InjectTXs(txs)

	case core.MsgTypeSetInjectDone2Shard:
		shard := shards_ref[id]
		shard.SetInjectTXDone()
    
    ...

    }
}
```

## result 模块
记录每笔交易的执行状态，输出时延、交易量、吞吐量等指标。

## shard 模块
管理区块链、交易池等数据结构。

## committee 模块
从shard中获取交易和状态，打包区块，执行交易，发送收据和区块等。

## client 模块
向分片注入交易。

## controller 模块
控制整个仿真流程
+ 读取配置文件
+ 从数据集读取交易数据
+ 初始化客户端、分片、委员会、信标链
+ 初始化 messageHub
+ 启动委员会和客户端线程
+ 启动记录线程
+ 关闭所有开启的线程

# todo
+ 已实现的部分：分片、客户端、跨分片交易、信标链、委员会
+ 待实现的部分：已实现部分的细化









