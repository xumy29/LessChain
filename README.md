# 写在前面
这份代码是ICBC2023的中稿论文 "W3Chain: A Layer2 Blockchain Defeating the Scalability Trilemma" 的仿真代码。在此基础上，对其进行了一些扩展。

# 配置运行
+ **数据集**： 以太坊第14920000个区块到第14960000个区块的数据（200w+条交易），存在本地未上传。

+ **运行模式**：分为 `debug` 模式和 `run` 模式。两个模式下分别读取的配置文件为 ".cfg/debug.json" 和 ".cfg/run.json"。

+ **参数文件设置**: 以 "./cfg/run.json" 为例
```json
{
    "LogLevel": 4,
    "LogFile": "newest_result.log",
    "IsProgressBar": true,
    "IsLogProgress": true,
    "LogProgressInterval": 20,

    "ClientNum": 1,

    "ShardNum": 2,
    "ShardSize": 4,

    "MaxTxNum": 5000,
    "InjectSpeed": 100,
    "RecommitInterval": 5,
    "MaxBlockTXSize": 100,

    "Height2Reconfig": 2000,

    "TbchainBlockIntervalSecs": 10,
    "TBChainHeight2Rollback": 2,

    "DatasetDir": "D:/project/blockChain/ethereum_data/14920000_14960000/ExternalTransactionItem.csv"
}
```


PS: RecommitInterval 即出块间隔，必须大于等于5s，否则会被强制设为5s。可在worker.go中更改此限制。


+ **编译运行**(run模式)
``` go
go build -o lessChain
./lessChain -m run
```
PS：windows系统要用exe后缀，即： go build -o lessChain.exe
    
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
	case core.MsgTypeCommitteeReply2Client:
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
一笔跨分片交易可能有多个状态，比如 [Cross1TXTypeSuccess, RollbackSuccess] 代表该交易先在委员会1上链，而后在委员会2上超时，进而被回滚。

## shard 模块
维护区块链，存储区块、状态，提供某笔交易在本分片区块链上链（proof of inclusion）或没有上链（proof of exclusion）的证据。

## committee 模块
维护交易池，从shard中获取状态，打包区块，执行交易，发送收据到客户端，发送区块到shard，发送信标到信标链，等等。
委员会会定期重组，其间组成各节点被随机打乱和重新分配。重组时丢弃交易池中的交易（轻量化是第一位）。

## beaconChain 模块
实现Layer1的信标链。有两种模式可供选择，分别是模拟的时间信标链和通过ganache部署的以太坊的私链。选择后者时需要先安装ganache并在debug.json中设置私链端口号。

## ganache模块
实现了与ganache私链交互的逻辑，如部署、调用合约、监听事件等。可以在beaconChain中调用。

## client 模块
向委员会注入交易，接收交易收据，当信标链对交易所在区块确认后，将交易状态记录到result模块，并对跨分片交易做下一步处理（发送后半部分或者回滚交易）。

## controller 模块
控制整个仿真流程
+ 读取配置文件
+ 从数据集读取交易数据
+ 初始化客户端、节点、分片、委员会、信标链
+ 初始化 messageHub
+ 启动委员会和客户端线程
+ 启动进度打印线程
+ 关闭所有开启的线程

# todo
merkle proof









