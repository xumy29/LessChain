#!/bin/bash

# 获取脚本所在的目录
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
GETH_DIR="$SCRIPT_DIR/eth_chain/geth-chain-data"

SHARD_NUM=2
SHARD_ALL_NODE_NUM=8

# 检查是否存在geth目录，如果存在则删除
if [ -d "$GETH_DIR/data/geth" ]; then
    rm -r $GETH_DIR/data/geth
fi

# 初始化
# 启动以太坊私链
echo "Starting Ethereum private chain..."
screen -d -m bash -c "cd $GETH_DIR && geth --datadir ./data init genesis.json && geth --datadir ./data --syncmode full --port 30310 --http --http.addr localhost --http.port 8545 --http.api personal,eth,net,web3,txpool,miner --ws --ws.port 8545 --allow-insecure-unlock --networkid 1337 -unlock '0x9128d0f6f5e04bd43305f7a323a67309c694a8f4' --password ./emptyPsw.txt --mine --miner.etherbase=0x9128d0f6f5e04bd43305f7a323a67309c694a8f4"
sleep 2

# 启动其他终端并运行相应的命令
echo "Starting booter..."
screen -d -m bash -c "cd $SCRIPT_DIR && go build -o ./lessChain && ./lessChain -r booter -S $SHARD_NUM"
sleep 5

echo "Starting client..."
screen -d -m bash -c "cd $SCRIPT_DIR && ./lessChain -r client -S $SHARD_NUM"
sleep 1


for ((j=0;j<$SHARD_NUM;j++));  
do
    for ((i=0;i<$SHARD_ALL_NODE_NUM;i++)); 
    do
        echo "Starting node S$j N$i..."
        screen -d -m bash -c "cd $SCRIPT_DIR && ./lessChain -r node -S $SHARD_NUM -s $j -n $i"
        sleep 0.2
    done
done


# #!/bin/bash
# # 该脚本不能在ssh连接的linux服务器上运行，可能原因是gnome-terminal需要图形会话界面

# # 获取脚本所在的目录
# SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
# GETH_DIR="$SCRIPT_DIR/eth_chain/geth-chain-data"

# # 跳转到指定目录
# # cd $GETH_DIR

# # 删除历史内容
# if [ -d "$GETH_DIR/data/geth" ]; then
#     rm -r $GETH_DIR/data/geth
# fi


# # 初始化
# # 启动以太坊私链
# gnome-terminal -- bash -c "cd $GETH_DIR && geth --datadir ./data init genesis.json && geth --datadir ./data --syncmode full --port 30310 --http --http.addr localhost --http.port 8545 --http.api personal,eth,net,web3,txpool,miner --ws --ws.port 8545 --allow-insecure-unlock --networkid 1337 -unlock '0x9128d0f6f5e04bd43305f7a323a67309c694a8f4' --password ./emptyPsw.txt --mine --miner.etherbase=0x9128d0f6f5e04bd43305f7a323a67309c694a8f4"
# if [ $? -ne 0 ]; then
#     echo "Error starting terminal for Ethereum private chain."
#     exit 1
# fi
# sleep 2

# # 启动其他终端并运行相应的命令
# gnome-terminal -- bash -c "cd $SCRIPT_DIR && go build -o ./lessChain && ./lessChain -r booter -S 2"
# if [ $? -ne 0 ]; then
#     echo "Error starting booter terminal."
#     exit 1
# fi
# sleep 5

# gnome-terminal -- bash -c "cd $SCRIPT_DIR && ./lessChain -r client -S 2"
# if [ $? -ne 0 ]; then
#     echo "Error starting client terminal."
#     exit 1
# fi
# sleep 1

# SHARD_NUM=2
# SHARD_ALL_NODE_NUM=4

# for ((j=0;j<$SHARD_NUM;j++));  
# do
#     for ((i=0;i<$SHARD_ALL_NODE_NUM;i++)); 
#     do
#         gnome-terminal -- bash -c "cd $SCRIPT_DIR && ./lessChain -r node -S $SHARD_NUM -s $j -n $i"
#         if [ $? -ne 0 ]; then
#             echo "Error starting node terminal with shard $j and node $i."
#             exit 1
#         fi
#     done
# done
