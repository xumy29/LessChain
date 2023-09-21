package messageHub

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"go-w3chain/core"
	"go-w3chain/log"
	"go-w3chain/result"
	"io"
	"net"
	"sync"
)

func listen(addr string, wg *sync.WaitGroup) {
	defer wg.Done()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("Error setting up listener", "err", err)
	}
	log.Info(fmt.Sprintf("start listening on %s", addr))
	listenConn = ln
	defer ln.Close()

	for {
		// // 超过时间限制没有收到新的连接则退出
		// ln.(*net.TCPListener).SetDeadline(time.Now().Add(10 * time.Second))
		conn, err := ln.Accept()
		if err != nil {
			log.Debug("Error accepting connection", "err", err)
			return
		}
		go handleConnection(conn, ln)

	}
}

func unpackMsg(packedMsg []byte) *core.Msg {
	var networkBuf bytes.Buffer
	networkBuf.Write(packedMsg)
	msgDec := gob.NewDecoder(&networkBuf)

	var msg core.Msg
	err := msgDec.Decode(&msg)
	if err != nil {
		log.Error("unpackMsgErr", "err", err, "msgBytes", packedMsg)
	}

	return &msg
}

func handleConnection(conn net.Conn, ln net.Listener) {
	defer conn.Close()

	// reader := bufio.NewReader(conn)

	for {
		// 先接收消息长度，再读消息
		lenBuf := make([]byte, 4)
		_, err := io.ReadFull(conn, lenBuf)
		if err != nil {
			if err.Error() == "EOF" {
				// 发送端主动关闭连接
				return
			}
			log.Error("Error reading from connection", "err", err)
		}
		length := int(binary.BigEndian.Uint32(lenBuf))
		packedMsg := make([]byte, length)
		_, err = io.ReadFull(conn, packedMsg)
		if err != nil {
			log.Error("Error reading from connection", "err", err)
		}

		msg := unpackMsg(packedMsg)
		switch msg.MsgType {
		// booter
		case ShardSendGenesis:
			exit := handleShardSendGenesis(msg.Data)
			if exit {
				ln.Close()
				return
			}
		case BooterSendContract:
			handleBooterSendContract(msg.Data)

		case ComGetHeight:
			handleComGetHeight(msg.Data, conn)

		case ComGetState:
			go handleComGetState(msg.Data)
		case ShardSendState:
			go handleShardSendState(msg.Data)

		case ClientSendTx:
			go handleClientSendTx(msg.Data)
		case ClientSetInjectDone:
			handleClientSetInjectDone(msg.Data)
		case ComSendTxReceipt:
			go handleComSendTxReceipt(msg.Data)

		case ComSendBlock:
			go handleComSendBlock(msg.Data)

		/////////////////////////
		//// pbft /////
		/////////////////////////
		case CPrePrepare, CPrepare, CCommit, CReply, CRequestOldrequest, CSendOldrequest:
			go handlePbftMsg(msg.Data, msg.MsgType)

		case NodeSendInfo:
			handleNodeSendInfo(msg.Data)

		default:
			log.Error("Unknown message type received", "msgType", msg.MsgType)
		}
	}
}

func handleClientSendTx(dataBytes []byte) {
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data []*core.Transaction
	err := dataDec.Decode(&data)
	if err != nil {
		log.Error("decodeDataErr", "err", err)
	}

	log.Info("Msg Received: ClientSendTx", "tx count", len(data))
	committee_ref.HandleClientSendtx(data)
}

func handleClientSetInjectDone(dataBytes []byte) {
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data *core.ClientSetInjectDone
	err := dataDec.Decode(&data)
	if err != nil {
		log.Error("decodeDataErr", "err", err)
	}

	log.Info("Msg Received: ClientSetInjectDone", "clientID", data.Cid)
	committee_ref.SetInjectTXDone(data.Cid)
}

func handleComGetHeight(dataBytes []byte, conn net.Conn) {
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data core.ComGetHeight
	err := dataDec.Decode(&data)
	if err != nil {
		log.Error("decodeDataErr", "err", err, "dataBytes", data)
	}

	log.Info("Msg Received: ComGetHeight", "data", data)

	// 直接通过这个连接回复请求方
	height := shard_ref.HandleComGetHeight(&data)
	var buf1 bytes.Buffer
	encoder := gob.NewEncoder(&buf1)
	err = encoder.Encode(height)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "data", data)
	}
	msgBytes := buf1.Bytes()

	// 前缀加上长度，防止粘包
	networkBuf := make([]byte, 4+len(msgBytes))
	binary.BigEndian.PutUint32(networkBuf[:4], uint32(len(msgBytes)))
	copy(networkBuf[4:], msgBytes)
	// 发送回复
	_, err = conn.Write(networkBuf)
	if err != nil {
		log.Error("WriteError", "err", err)
	}

}

func handleComGetState(dataBytes []byte) {
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data core.ComGetState
	err := dataDec.Decode(&data)
	if err != nil {
		log.Error("decodeDataErr", "err", err, "dataBytes", data)
	}

	log.Info("Msg Received: ComGetState", "addr count", len(data.AddrList))
	shard_ref.HandleComGetState(&data)
}

func handleShardSendState(dataBytes []byte) {
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data core.ShardSendState
	err := dataDec.Decode(&data)
	if err != nil {
		log.Error("decodeDataErr", "err", err, "dataBytes", data)
	}

	log.Info("Msg Received: ShardSendState", "addr count", len(data.AccountData))
	committee_ref.HandleShardSendState(&data)
}

func handleBooterSendContract(dataBytes []byte) {
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data core.BooterSendContract
	err := dataDec.Decode(&data)
	if err != nil {
		log.Error("decodeDataErr", "err", err, "dataBytes", data)
	}

	log.Info("Msg Received: BooterSendContract", "data", data)

	if node_ref != nil {
		// 注意是节点处理而不是分片或委员会处理
		node_ref.HandleBooterSendContract(&data)
	}
	if client_ref != nil {
		client_ref.HandleBooterSendContract(&data)
	}
	tbChain_ref.HandleBooterSendContract(&data)

}

func handleComSendBlock(dataBytes []byte) {
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data core.ComSendBlock
	err := dataDec.Decode(&data)
	if err != nil {
		log.Error("decodeDataErr", "err", err, "dataBytes", data)
	}

	log.Info("Msg Received: ComSendBlock", "tx count", len(data.Transactions))

	shard_ref.HandleComSendBlock(&data)
}

//////////////////////////////////////////////////
////  client  ////
//////////////////////////////////////////////////

func handleComSendTxReceipt(dataBytes []byte) {
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data []*result.TXReceipt
	err := dataDec.Decode(&data)
	if err != nil {
		log.Error("decodeDataErr", "err", err, "dataBytes", data)
	}

	log.Info("Msg Received: ComSendTxReceipt", "tx count", len(data))

	client_ref.HandleComSendTxReceipt(data)
}

//////////////////////////////////////////////////
////  booter  ////
//////////////////////////////////////////////////

func handleShardSendGenesis(dataBytes []byte) (exit bool) {
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data core.ShardSendGenesis
	err := dataDec.Decode(&data)
	if err != nil {
		log.Error("decodeDataErr", "err", err, "dataBytes", data)
	}

	log.Info("Msg Received: ShardSendGenesis")

	exit = booter_ref.HandleShardSendGenesis(&data)
	return exit
}

//////////////////////////////////////////////////
////  pbft module  ////
//////////////////////////////////////////////////
func handlePbftMsg(dataBytes []byte, dataType string) {
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	switch dataType {
	case CPrePrepare:
		var data core.PrePrepare
		err := dataDec.Decode(&data)
		if err != nil {
			log.Error("decodeDataErr", "err", err, "dataBytes", data)
		}
		log.Info(fmt.Sprintf("Msg Received: %s ComID: %v", dataType, pbftNode_ref.ComID))
		pbftNode_ref.HandlePrePrepare(&data)
	case CPrepare:
		var data core.Prepare
		err := dataDec.Decode(&data)
		if err != nil {
			log.Error("decodeDataErr", "err", err, "dataBytes", data)
		}
		log.Info(fmt.Sprintf("Msg Received: %s ComID: %v from nodeID: %v", dataType, pbftNode_ref.ComID, data.SenderInfo.NodeID))
		pbftNode_ref.HandlePrepare(&data)
	case CCommit:
		var data core.Commit
		err := dataDec.Decode(&data)
		if err != nil {
			log.Error("decodeDataErr", "err", err, "dataBytes", data)
		}
		log.Info(fmt.Sprintf("Msg Received: %s ComID: %v from nodeID: %v", dataType, pbftNode_ref.ComID, data.SenderInfo.NodeID))
		pbftNode_ref.HandleCommit(&data)
	case CReply:
		var data core.Reply
		err := dataDec.Decode(&data)
		if err != nil {
			log.Error("decodeDataErr", "err", err, "dataBytes", data)
		}
		log.Info(fmt.Sprintf("Msg Received: %s ComID: %v from nodeID: %v", dataType, pbftNode_ref.ComID, data.SenderInfo.NodeID))
		pbftNode_ref.HandleReply(&data)
	case CRequestOldrequest:
		var data core.RequestOldMessage
		err := dataDec.Decode(&data)
		if err != nil {
			log.Error("decodeDataErr", "err", err, "dataBytes", data)
		}
		log.Info(fmt.Sprintf("Msg Received: %s ComID: %v from nodeID: %v", dataType, pbftNode_ref.ComID, data.SenderInfo.NodeID))
		pbftNode_ref.HandleRequestOldSeq(&data)
	case CSendOldrequest:
		var data core.SendOldMessage
		err := dataDec.Decode(&data)
		if err != nil {
			log.Error("decodeDataErr", "err", err, "dataBytes", data)
		}
		log.Info(fmt.Sprintf("Msg Received: %s ComID: %v from nodeID: %v", dataType, pbftNode_ref.ComID, data.SenderInfo.NodeID))
		pbftNode_ref.HandleSendOldSeq(&data)
	}

}

func handleNodeSendInfo(dataBytes []byte) {
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data core.NodeSendInfo
	err := dataDec.Decode(&data)
	if err != nil {
		log.Error("decodeDataErr", "err", err, "dataBytes", data)
	}

	log.Info("Msg Received: NodeSendInfo")

	node_ref.HandleNodeSendInfo(&data)
}
