package messageHub

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"go-w3chain/cfg"
	"go-w3chain/core"
	"go-w3chain/log"
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

	var networkBuf bytes.Buffer
	msgEnc := gob.NewEncoder(&networkBuf)
	err := msgEnc.Encode(msg)
	if err != nil {
		log.Error("gobEncodeErr", "err", err, "msg", msg)
	}

	return networkBuf.Bytes()
}

func clientInjectTx2Com(comID uint32, msg interface{}) {

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
}
