package messageHub

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"go-w3chain/beaconChain"
	"go-w3chain/cfg"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"io"
	"math/big"
	"net"
)

func dial(addr string) net.Conn {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		// log.Error("DialTCPError", "target_addr", addr, "err", err)
		panic(err)
	}
	return conn
}

func packMsg(msgType string, data []byte) []byte {
	msg := &core.Msg{
		MsgType: msgType,
		Data:    data,
	}

	var buf bytes.Buffer
	msgEnc := gob.NewEncoder(&buf)
	err := msgEnc.Encode(msg)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "msg", msg)
	}

	msgBytes := buf.Bytes()

	// 前缀加上长度，防止粘包
	networkBuf := make([]byte, 4+len(msgBytes))
	binary.BigEndian.PutUint32(networkBuf[:4], uint32(len(msgBytes)))
	copy(networkBuf[4:], msgBytes)

	return networkBuf
}

func comGetHeightFromShard(shardID uint32, msg interface{}) *big.Int {
	data := msg.(*core.ComGetHeight)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}

	// 序列化后的消息
	msg_bytes := packMsg("ComGetHeight", buf.Bytes())

	conn, ok := conns2Shard.Get(shardID)
	if !ok {
		conn = dial(cfg.NodeTable[shardID][0])
		conns2Shard.Add(shardID, conn)
	}
	_, err = conn.Write(msg_bytes)
	if err != nil {
		log.Error("WriteError", "err", err)
	}
	log.Info("Msg Sent: ComGetHeight", "data", "empty data")

	// 等待回复

	// 首先读取消息长度的四个字节
	lengthBuf := make([]byte, 4)
	_, err = io.ReadFull(conn, lengthBuf)
	if err != nil {
		log.Error("ReadLengthError", "err", err)
	}
	// 解析这四个字节为int32来获取消息长度
	msgLength := int(binary.BigEndian.Uint32(lengthBuf))

	// 根据消息长度分配缓冲区
	msgBuf := make([]byte, msgLength)
	_, err = io.ReadFull(conn, msgBuf)
	if err != nil {
		log.Error("ReadMsgError", "err", err)
	}

	height := new(big.Int)
	decodeBuf := bytes.NewReader(msgBuf)
	decoder := gob.NewDecoder(decodeBuf)
	err = decoder.Decode(height)
	if err != nil {
		log.Error("Failed to decode height using gob", "err", err)
	}

	log.Info("Msg Response Received: ComGetHeight", "height", height)

	return height
}

/* 分片向一个中心化节点发送创世区块信标和初始账户地址
该消息只进行一次，不需通过长连接发送
*/
func shardSendGenesis(msg interface{}) {
	data := msg.(*core.ShardSendGenesis)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}

	// 序列化后的消息
	msg_bytes := packMsg("ShardSendGenesis", buf.Bytes())

	// 连接到目标地址
	conn, err := net.Dial("tcp", data.Target_nodeAddr)
	if err != nil {
		log.Error("DialError", "err", err)
	}
	defer conn.Close()

	// 写入消息
	_, err = conn.Write(msg_bytes)
	if err != nil {
		log.Error("WriteError", "err", err)
	}

	log.Info("Msg Sent: ShardSendGenesis", "data", data)
}

func clientInjectTx2Com(comID uint32, msg interface{}) {
	data := msg.([]*core.Transaction)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}

	// 序列化后的消息
	msg_bytes := packMsg("ClientSendTx", buf.Bytes())

	conn, ok := conns2Com.Get(comID)
	if !ok {
		conn = dial(cfg.NodeTable[comID][0])
		conns2Com.Add(comID, conn)
	}
	writer := bufio.NewWriter(conn)
	writer.Write(msg_bytes)
	writer.Flush()

	log.Info("Msg Sent: ClientSendTx", "targetComID", comID, "tx count", len(data))
}

func clientSetInjectDone2Nodes(cid uint32) {
	data := &core.ClientSetInjectDone{Cid: cid}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}

	// 序列化后的消息
	msg_bytes := packMsg("ClientSetInjectDone", buf.Bytes())

	// todo: 修改成向所有节点发送
	// 向每个分片的leader节点发送合约地址等信息
	var i uint32
	for i = 0; i < uint32(shardNum); i++ {
		conn, ok := conns2Shard.Get(i)
		if !ok {
			conn = dial(cfg.NodeTable[i][0])
			conns2Shard.Add(i, conn)
		}
		_, err := conn.Write(msg_bytes)
		if err != nil {
			panic(err)
		}
		conn.Close()
	}

	log.Info("Msg Sent: ClientSetInjectDone", "clientID", cid)
}

func comGetStateFromShard(shardID uint32, msg interface{}) {
	data := msg.(*core.ComGetState)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}

	// 序列化后的消息
	msg_bytes := packMsg("ComGetState", buf.Bytes())

	conn, ok := conns2Shard.Get(shardID)
	if !ok {
		conn = dial(cfg.NodeTable[shardID][0])
		conns2Shard.Add(shardID, conn)
	}
	writer := bufio.NewWriter(conn)
	writer.Write(msg_bytes)
	writer.Flush()

	log.Info("Msg Sent: ComGetState", "addr count", len(data.AddrList))
}

func shardSendStateToCom(comID uint32, msg interface{}) {
	data := msg.(*core.ShardSendState)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}

	// 序列化后的消息
	msg_bytes := packMsg("ShardSendState", buf.Bytes())

	conn, ok := conns2Shard.Get(comID)
	if !ok {
		conn = dial(cfg.NodeTable[comID][0])
		conns2Shard.Add(comID, conn)
	}
	writer := bufio.NewWriter(conn)
	writer.Write(msg_bytes)
	writer.Flush()

	log.Info("Msg Sent: ShardSendState", "addr count", len(data.AccountData))
}

func comSendBlock2Shard(shardID uint32, msg interface{}) {
	data := msg.(*core.ComSendBlock)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}

	// 序列化后的消息
	msg_bytes := packMsg("ComSendBlock", buf.Bytes())

	conn, ok := conns2Shard.Get(shardID)
	if !ok {
		conn = dial(cfg.NodeTable[shardID][0])
		conns2Shard.Add(shardID, conn)
	}
	writer := bufio.NewWriter(conn)
	writer.Write(msg_bytes)
	writer.Flush()

	log.Info("Msg Sent: ComSendBlock", "toShardID", shardID, "tx count", len(data.Transactions))
}

func comSendReply2Client(clientID uint32, msg interface{}) {
	data := msg.([]*result.TXReceipt)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}

	// 序列化后的消息
	msg_bytes := packMsg("ComSendTxReceipt", buf.Bytes())

	conn, ok := conns2Client.Get(clientID)
	if !ok {
		conn = dial(cfg.ClientTable[clientID])
		conns2Client.Add(clientID, conn)
	}
	writer := bufio.NewWriter(conn)
	writer.Write(msg_bytes)
	writer.Flush()

	log.Info("Msg Sent: ComSendTxReceipt", "toClientID", clientID, "tx count", len(data))
}

// 调用beaconChain包，再通过ethClient与ethchain交互
// 或者beaconChain监听到的ethchain事件，发送给客户端、委员会

func comAddTb2TBChain(msg interface{}) {
	data := msg.(*beaconChain.SignedTB)
	tbChain_ref.AddTimeBeacon(data)
}

func comGetLatestBlock(comID uint32, callback func(...interface{})) {
	hash, height := tbChain_ref.GetEthChainLatestBlockHash(comID)
	callback(hash, height)
}

func tbChainPushBlock2Client(msg interface{}) {
	if client_ref == nil {
		return
	}
	data := msg.(*beaconChain.TBBlock)
	client_ref.AddTBs(data)
}

//////////////////////////////////////////////////
////  booter  ////
//////////////////////////////////////////////////

func booterSendContract(msg interface{}) {
	data := msg.(*core.BooterSendContract)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}

	// 序列化后的消息
	msg_bytes := packMsg("BooterSendContract", buf.Bytes())

	// todo: 修改成向所有节点发送
	// 向每个分片的leader节点发送合约地址等信息
	var i uint32
	for i = 0; i < uint32(shardNum); i++ {
		conn, ok := conns2Shard.Get(i)
		if !ok {
			conn = dial(cfg.NodeTable[i][0])
			conns2Shard.Add(i, conn)
		}
		_, err := conn.Write(msg_bytes)
		if err != nil {
			panic(err)
		}
		conn.Close()
	}

	// 向客户端发送合约地址等信息
	for i = 0; i < uint32(clientNum); i++ {
		conn, ok := conns2Client.Get(i)
		if !ok {
			conn = dial(cfg.ClientTable[i])
			conns2Client.Add(i, conn)
		}
		_, err := conn.Write(msg_bytes)
		if err != nil {
			panic(err)
		}
		conn.Close()
	}

	log.Info("Msg Sent: BooterSendContract", "data", data)

}
