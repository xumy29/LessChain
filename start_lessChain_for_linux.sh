#!/bin/bash

# 获取脚本所在的目录
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
GETH_DIR="$SCRIPT_DIR/eth_chain/geth-chain-data"

# 从配置文件中读取值
CONFIG_FILE="$SCRIPT_DIR/cfg/debug.json"
SHARD_NUM=$(jq .ShardNum $CONFIG_FILE)
SHARD_ALL_NODE_NUM=$(jq .ComAllNodeNum $CONFIG_FILE)
SHARD_START_INDEX=$(jq .ShardStartIndex $CONFIG_FILE)
SHARD_END_INDEX=$(jq .ShardEndIndex $CONFIG_FILE)
ROLE=$(jq -r .Role $CONFIG_FILE)

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
    screen -d -m bash -c "cd $SCRIPT_DIR && go build -o ./lessChain && ./lessChain -r booter -S $SHARD_NUM"
    sleep 5

    echo "Starting client..."
    screen -d -m bash -c "cd $SCRIPT_DIR && ./lessChain -r client -S $SHARD_NUM"
    sleep 1
else
    # 启动分片节点
    for ((j=$SHARD_START_INDEX;j<=$SHARD_END_INDEX;j++));
    do
        for ((i=0;i<$SHARD_ALL_NODE_NUM;i++));
        do
            echo "Starting node S$j N$i..."
            screen -d -m bash -c "cd $SCRIPT_DIR && ./lessChain -r node -S $SHARD_NUM -s $j -n $i"
            sleep 0.2
        done
    done
fi



