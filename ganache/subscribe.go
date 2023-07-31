package ganache

import (
	"encoding/json"
	"fmt"
	"go-w3chain/log"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
)

type WebSocketResponse struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Subscription string `json:"subscription"`
		Result       struct {
			LogIndex         string   `json:"logIndex"`
			TransactionIndex string   `json:"transactionIndex"`
			TransactionHash  string   `json:"transactionHash"`
			BlockHash        string   `json:"blockHash"`
			BlockNumber      string   `json:"blockNumber"`
			Address          string   `json:"address"`
			Data             string   `json:"data"`
			Topics           []string `json:"topics"`
			Type             string   `json:"type"`
		} `json:"result"`
	} `json:"params"`
}

type Event struct {
	Msg     string
	ShardID uint32
	Height  uint64
}

func SubscribeEvents(port int, contractAddr common.Address, eventChannel chan *Event) {
	// WebSocket 连接地址
	url := fmt.Sprintf("ws://localhost:%d", port)

	// 创建 WebSocket 连接
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Error("Failed to connect to Ganache WebSocket:", "err", err)
		panic(err)
	}
	defer conn.Close()

	// 订阅合约事件
	subscribeRequest := []byte(fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "eth_subscribe",
		"params": ["logs", {
			"address": "%v"
		}],
		"id": 1
	}`, contractAddr.Hex()))

	err = conn.WriteMessage(websocket.TextMessage, subscribeRequest)
	if err != nil {
		log.Error("Failed to send subscription request:", "err", err)
		panic(err)
	}

	// 处理事件通知
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Error("Failed to read message from Ganache WebSocket:", "err", err)
		}

		// fmt.Println("Received message:", string(message))

		// 在这里处理事件通知，解析事件数据等
		if string(message)[2:4] == "id" {
			continue
		}
		var response WebSocketResponse
		err = json.Unmarshal(message, &response)
		if err != nil {
			fmt.Println("Failed to unmarshal JSON:", err)
			log.Debug("Failed to unmarshal JSON", "err", err)
			continue
		}

		data := response.Params.Result.Data
		// fmt.Println("Data:", data)
		event := handleMessage(data)
		if event.Msg == "addTB" {
			eventChannel <- event
		}
	}
}

func handleMessage(data string) *Event {
	eventABI := `
	[
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": false,
					"internalType": "string",
					"name": "message",
					"type": "string"
				},
				{
					"indexed": false,
					"internalType": "uint32",
					"name": "shardID",
					"type": "uint32"
				},
				{
					"indexed": false,
					"internalType": "uint64",
					"name": "height",
					"type": "uint64"
				}
			],
			"name": "LogMessage",
			"type": "event"
		}
	]
	`
	abi, err := abi.JSON(strings.NewReader(eventABI))
	if err != nil {
		panic(err)
	}

	decodedData := make(map[string]interface{})
	err = abi.UnpackIntoMap(decodedData, "LogMessage", common.FromHex(data))
	if err != nil {
		panic(err)
	}

	// 读取数据前可以先打印看看有哪些字段
	// fmt.Printf("decodedData: %v\n", decodedData)

	message := decodedData["message"].(string)
	shardID := decodedData["shardID"].(uint32)
	height := decodedData["height"].(uint64)

	// addr := common.Address{}
	// if decodedData["address"] != nil {
	// 	addr = decodedData["address"].(common.Address)
	// }
	// addr := decodedData["addr"].([32]byte)

	fmt.Printf("Message: %s\t", message)
	fmt.Printf("ShardID: %d\t", shardID)
	fmt.Printf("Height: %d\n", height)
	// fmt.Printf("Address: %v\n", addr)

	return &Event{
		Msg:     message,
		ShardID: shardID,
		Height:  height,
	}
}
