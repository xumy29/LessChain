package messageHub

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
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

	// 从分片的leader节点处获取
	addr := cfg.NodeTable[shardID][0]
	conn, ok := conns2Node.Get(addr)
	if !ok {
		conn = dial(addr)
		conns2Node.Add(addr, conn)
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

	// 发送给委员会的leader即可
	addr := cfg.ComNodeTable[comID][0]
	conn, ok := conns2Node.Get(addr)
	if !ok {
		conn = dial(addr)
		conns2Node.Add(addr, conn)
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

	// 向所有节点发送合约地址等信息
	var i, j uint32
	for i = 0; i < uint32(shardNum); i++ {
		for j = 0; j < uint32(shardSize); j++ {
			addr := cfg.NodeTable[i][j]
			conn, ok := conns2Node.Get(addr)
			if !ok {
				conn = dial(addr)
				conns2Node.Add(addr, conn)
			}
			_, err := conn.Write(msg_bytes)
			if err != nil {
				panic(err)
			}
			conn.Close()
		}
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

	// 从分片的leader节点获取
	addr := cfg.NodeTable[shardID][0]
	conn, ok := conns2Node.Get(addr)
	if !ok {
		conn = dial(addr)
		conns2Node.Add(addr, conn)
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
	// 只发送给委员会的leader节点
	addr := cfg.ComNodeTable[comID][0]
	conn, ok := conns2Node.Get(addr)
	if !ok {
		conn = dial(addr)
		conns2Node.Add(addr, conn)
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

	// 只发送给分片的leader节点
	addr := cfg.NodeTable[shardID][0]
	conn, ok := conns2Node.Get(addr)
	if !ok {
		conn = dial(addr)
		conns2Node.Add(addr, conn)
	}
	writer := bufio.NewWriter(conn)
	writer.Write(msg_bytes)
	writer.Flush()

	log.Info("Msg Sent: ComSendBlock", "toShardID", shardID, "tx count", len(data.Block.Transactions))
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

	addr := cfg.ClientTable[clientID]
	conn, ok := conns2Node.Get(addr)
	if !ok {
		conn = dial(addr)
		conns2Node.Add(addr, conn)
	}
	writer := bufio.NewWriter(conn)
	writer.Write(msg_bytes)
	writer.Flush()

	log.Info("Msg Sent: ComSendTxReceipt", "toClientID", clientID, "tx count", len(data))
}

// 调用beaconChain包，再通过ethClient与ethchain交互
// 或者beaconChain监听到的ethchain事件，发送给客户端、委员会

func comAddTb2TBChain(msg interface{}) {
	data := msg.(*core.SignedTB)
	tbChain_ref.AddTimeBeacon(data)
}

func getEthLatestBlock(callback func(...interface{})) {
	hash, height := tbChain_ref.GetEthChainLatestBlockHash()
	callback(hash, height)
}

func getEthBlock(msg interface{}, callback func(...interface{})) {
	height := msg.(uint64)
	hash, got_height := tbChain_ref.GetEthChainBlockHash(height)
	callback(hash, got_height)
}

func tbChainPushBlock2Client(msg interface{}) {
	if client_ref == nil {
		return
	}
	data := msg.(*beaconChain.TBBlock)
	client_ref.AddTBs(data)
}

func comLeaderInitMultiSign(comID uint32, msg interface{}) {
	data := msg.(*core.ComLeaderInitMultiSign)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}

	// 序列化后的消息
	msg_bytes := packMsg(LeaderInitMultiSign, buf.Bytes())

	// 向委员会中的所有节点发送（包括自己）
	var i uint32
	for i = 0; i < uint32(shardSize); i++ {
		addr := cfg.ComNodeTable[comID][i]
		conn, ok := conns2Node.Get(addr)
		if !ok {
			conn = dial(addr)
			conns2Node.Add(addr, conn)
		}
		_, err := conn.Write(msg_bytes)
		if err != nil {
			log.Debug(fmt.Sprintf("write err: %v", err))
		}
	}
	log.Info("Msg Sent: comLeaderInitMultiSign", "comID", comID)
}

func sendMultiSignReply(comID uint32, msg interface{}) {
	data := msg.(*core.MultiSignReply)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}

	// 序列化后的消息
	msg_bytes := packMsg(MultiSignReply, buf.Bytes())

	// 向委员会的leader节点发送

	addr := cfg.ComNodeTable[comID][0]
	conn, ok := conns2Node.Get(addr)
	if !ok {
		conn = dial(addr)
		conns2Node.Add(addr, conn)
	}
	_, err = conn.Write(msg_bytes)
	if err != nil {
		panic(err)
	}

	log.Info("Msg Sent: multiSignReply", "comID", comID)
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
		addr := cfg.ComNodeTable[i][0]
		conn, ok := conns2Node.Get(addr)
		if !ok {
			conn = dial(addr)
		}
		_, err := conn.Write(msg_bytes)
		if err != nil {
			panic(err)
		}
		conn.Close()
	}

	// 向客户端发送合约地址等信息
	for i = 0; i < uint32(clientNum); i++ {
		addr := cfg.ClientTable[i]
		conn, ok := conns2Node.Get(addr)
		if !ok {
			conn = dial(addr)
		}
		_, err := conn.Write(msg_bytes)
		if err != nil {
			panic(err)
		}
		conn.Close()
	}

	log.Info("Msg Sent: BooterSendContract", "data", data)

}

/////////////////////
///// pbft //////////
/////////////////////
func sendPbftMsg(comID uint32, msg interface{}, msgType string) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	var err error

	switch msgType {
	case CPrePrepare:
		data := msg.(*core.PrePrepare)
		err = enc.Encode(data)
	case CPrepare:
		data := msg.(*core.Prepare)
		err = enc.Encode(data)
	case CCommit:
		data := msg.(*core.Commit)
		err = enc.Encode(data)
	case CReply:
		data := msg.(*core.Reply)
		err = enc.Encode(data)
	case CRequestOldrequest:
		data := msg.(*core.RequestOldMessage)
		err = enc.Encode(data)
	case CSendOldrequest:
		data := msg.(*core.SendOldMessage)
		err = enc.Encode(data)
	default:
		log.Error("unknown pbft msg type", "type", msgType)
	}

	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data(interface{})", msg)
	}

	// 序列化后的消息
	msg_bytes := packMsg(msgType, buf.Bytes())

	var i uint32
	nodeAddr := pbftNode_ref.NodeInfo.NodeAddr
	for i = 0; i < uint32(shardSize); i++ {
		if msgType == CReply && i > 0 { // reply只需发给leader
			return
		}
		addr := cfg.ComNodeTable[comID][i]
		if addr == nodeAddr {
			continue // 不用发给自己
		}
		conn, ok := conns2Node.Get(addr)
		if !ok {
			conn = dial(addr)
			conns2Node.Add(addr, conn)
		}
		writer := bufio.NewWriter(conn)
		writer.Write(msg_bytes)
		writer.Flush()

		log.Info(fmt.Sprintf("Msg Sent: %s ComID: %v, to nodeID: %v", msgType, comID, i))
	}
}

func sendNodeInfo(comID uint32, msg interface{}) {
	data := msg.(*core.NodeSendInfo)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}

	// 序列化后的消息
	msg_bytes := packMsg(NodeSendInfo, buf.Bytes())

	addr := cfg.ComNodeTable[comID][0]
	conn, ok := conns2Node.Get(addr)
	if !ok {
		conn = dial(addr)
		conns2Node.Add(addr, conn)
	}
	writer := bufio.NewWriter(conn)
	writer.Write(msg_bytes)
	writer.Flush()

	log.Info("Msg Sent: NodeSendInfo", "ComID", comID, "to nodeID", 0)
}
