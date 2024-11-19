#!/bin/bash

# 获取脚本所在的目录
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
GETH_DIR="$SCRIPT_DIR/eth_chain/geth-chain-data"

# 从配置文件中读取值
CONFIG_FILE="$SCRIPT_DIR/cfg/debug.json"
SHARD_NUM=$(jq .ShardNum $CONFIG_FILE)
SHARD_ALL_NODE_NUM=$(jq .ComAllNodeNum $CONFIG_FILE)
MACHINE_NUM=$(jq .MachineNum $CONFIG_FILE)
SHARD_START_INDEX=$(jq .ShardStartIndex $CONFIG_FILE)
ROLE=$(jq -r .Role $CONFIG_FILE)
# 如果系统上的go版本大于等于1.17，则直接使用，否则安装go1.21
## 检查系统是否安装 Go 以及 Go 版本是否大于等于 1.17
if ! go version &> /dev/null || [[ "$(go version | awk '{print $3}')" < "go1.17" ]]; then
    # 设置 Go 安装目录
    GO_DIR=$HOME/xmyWorkspace/software

    echo "系统没有安装 Go 或 Go 版本小于 1.17，开始安装 Go 1.21.1..."

    # 下载 Go 1.21.1
    wget https://mirrors.aliyun.com/golang/go1.21.1.linux-amd64.tar.gz -P /tmp

    # 解压 Go 到指定目录
    mkdir -p $GO_DIR
    sudo tar -C $GO_DIR -xzf /tmp/go1.21.1.linux-amd64.tar.gz

    # 清理下载的文件
    rm /tmp/go1.21.1.linux-amd64.tar.gz

    # 设置 GO 环境变量
    echo "export PATH=$GO_DIR/go/bin:$PATH" >> ~/.bashrc

    # 重新加载 shell 配置
    source ~/.bashrc
fi

# 确认 Go 版本
echo "Go 版本：$(go version)"
GO=go

if [ "$ROLE" != "node" ]
then
    # 启动geth、booteer和client
    # 检查是否存在geth目录，如果存在则删除
    if [ -d "$GETH_DIR/data/geth" ]; then
        rm -r $GETH_DIR/data/geth
    fi

    # 初始化
    # 启动以太坊私链
    echo "Starting Ethereum private chain..."
    screen -d -m bash -c "cd $GETH_DIR && geth --datadir ./data init genesis.json && geth --verbosity 5 --datadir ./data  --syncmode full --port 30310 --http --http.addr "0.0.0.0" --http.corsdomain '*' --http.port 8545 --http.api personal,eth,net,web3,txpool,miner --ws --ws.addr "0.0.0.0" --ws.port 8545 --ws.origins '*' --allow-insecure-unlock --networkid 1337 -unlock '0x9128d0f6f5e04bd43305f7a323a67309c694a8f4' --password ./emptyPsw.txt --mine --miner.etherbase=0x9128d0f6f5e04bd43305f7a323a67309c694a8f4 > geth.log 2>&1"
    sleep 2

    # 启动其他终端并运行相应的命令
    echo "Starting booter..."
    screen -d -m bash -c "$GO build -o ./lessChain && ./lessChain -r booter -S $SHARD_NUM; tail -f /dev/null"
    sleep 5

    echo "Starting client..."
    screen -d -m bash -c "./lessChain -r client -S $SHARD_NUM; tail -f /dev/null"
    sleep 1
else
    $GO build -o ./lessChain
    sleep 5
    # 启动分片节点
    for ((j=0;j<$((SHARD_NUM / MACHINE_NUM));j++));
    do
        shardIndex=$((SHARD_START_INDEX + MACHINE_NUM * j))
        echo "Starting node S$shardIndex N0..."
        screen -d -m bash -c "./lessChain -r node -S $SHARD_NUM -s $shardIndex -n 0; tail -f /dev/null"
        sleep 2
        for ((i=1;i<$SHARD_ALL_NODE_NUM;i++));
        do
            echo "Starting node S$shardIndex N$i..."
            screen -d -m bash -c "./lessChain -r node -S $SHARD_NUM -s $shardIndex -n $i; tail -f /dev/null"
            sleep 0.2
        done
    done
fi



