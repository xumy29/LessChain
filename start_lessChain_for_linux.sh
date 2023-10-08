#!/bin/bash

# 获取脚本所在的目录
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
GETH_DIR="$SCRIPT_DIR/eth_chain/geth-chain-data"

# 跳转到指定目录
cd $GETH_DIR

# 删除历史内容
rm -r ./data/geth 

# 初始化
geth --datadir ./data init genesis.json
sleep 2

# 启动以太坊私链
gnome-terminal -- bash -c "cd $GETH_DIR && geth --datadir ./data --syncmode full --port 30310 --http --http.addr localhost --http.port 8545 --http.api personal,eth,net,web3,txpool,miner --ws --ws.port 8545 --allow-insecure-unlock --networkid 1337 -unlock '0x9128d0f6f5e04bd43305f7a323a67309c694a8f4' --password ./emptyPsw.txt --mine --miner.etherbase=0x9128d0f6f5e04bd43305f7a323a67309c694a8f4"
sleep 2

# 启动其他终端并运行相应的命令
gnome-terminal -- bash -c "cd $SCRIPT_DIR && go build -o ./lessChain && ./lessChain -r booter -S 2"
sleep 2

gnome-terminal -- bash -c "cd $SCRIPT_DIR && ./lessChain -r client -S 2"
sleep 1

SHARD_NUM=2
SHARD_ALL_NODE_NUM=8

for ((j=0;j<$SHARD_NUM;j++));  
do
    for ((i=0;i<$SHARD_ALL_NODE_NUM;i++)); 
    do
        gnome-terminal -- bash -c "cd $SCRIPT_DIR && ./lessChain -r node -S $SHARD_NUM -s $j -n $i"
    done
done
