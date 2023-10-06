#!/bin/bash
## 启动lessChain的脚本
## 注：针对Windows的部分使用了osascript，这实际上是macOS特有的，并不能在Windows上运行。
## 如果需要在Windows上执行类似的自动化任务，需要使用其他工具或方法。

# 一个简单的函数，计算每个新窗口的位置。
get_window_position() {
    local index=$1
    local offset=30
    local x=$((index * offset))
    local y=$((index * offset))
    echo "$x,$y"
}

# 使用循环启动终端并设置每个窗口的位置
counter=0

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

# 判断操作系统并执行相应的命令
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    osascript -e 'tell app "Terminal" to do script "cd '$GETH_DIR' && geth --datadir ./data --syncmode '\''full'\'' --port 30310 --http --http.addr '\''localhost'\'' --http.port 8545 --http.api '\''personal,eth,net,web3,txpool,miner'\'' --ws --ws.port 8545 --allow-insecure-unlock --networkid 1337 -unlock '\''0x9128d0f6f5e04bd43305f7a323a67309c694a8f4'\'' --password ./emptyPsw.txt --mine --miner.etherbase=0x9128d0f6f5e04bd43305f7a323a67309c694a8f4"'
    position=$(get_window_position 20)
    osascript -e "tell application \"Terminal\" to set position of front window to {$position}"
elif [[ "$OSTYPE" == "cygwin" ]] || [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "win32" ]]; then
    # Windows
    osascript -e 'tell app "Terminal" to do script "cd '$GETH_DIR' && geth --datadir ./data --syncmode full --port 30310 --http --http.addr localhost --http.port 8545 --http.api personal,eth,net,web3,txpool,miner --ws --ws.port 8545 --allow-insecure-unlock --networkid 1337 -unlock 0x9128d0f6f5e04bd43305f7a323a67309c694a8f4 --password ./emptyPsw.txt --mine --miner.etherbase=0x9128d0f6f5e04bd43305f7a323a67309c694a8f4"'
else
    echo "Unknown OS type: $OSTYPE"
    exit 1
fi
sleep 2

# 启动终端1
osascript -e 'tell app "Terminal" to do script "cd '$SCRIPT_DIR' && go build -o ./lessChain && ./lessChain -r booter -S 2"'
sleep 2
((counter++))
position=$(get_window_position $counter)
osascript -e "tell application \"Terminal\" to set position of front window to {$position}"


# 启动终端2
osascript -e 'tell app "Terminal" to do script "cd '$SCRIPT_DIR' && ./lessChain -r client -S 2"'
sleep 1
((counter++))
position=$(get_window_position $counter)
osascript -e "tell application \"Terminal\" to set position of front window to {$position}"


# 启动4个终端，运行节点0~3
for i in {0..3}
do
    osascript -e 'tell app "Terminal" to do script "cd '$SCRIPT_DIR' && ./lessChain -r node -S 2 -s 0 -n '$i'"'
    ((counter++))
    position=$(get_window_position $counter)
    osascript -e "tell application \"Terminal\" to set position of front window to {$position}"

done

# 启动4个终端，运行节点0~3
for i in {0..3}
do
    osascript -e 'tell app "Terminal" to do script "cd '$SCRIPT_DIR' && ./lessChain -r node -S 2 -s 1 -n '$i'"'
    ((counter++))
    position=$(get_window_position $counter)
    osascript -e "tell application \"Terminal\" to set position of front window to {$position}"
done
