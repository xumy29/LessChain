package messageHub

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"go-w3chain/core"
	"go-w3chain/log"
	"net"
)

func init() {
	gob.Register(core.Msg{})
}

func listen(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("Error setting up listener", "err", err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Error("Error accepting connection", "err", err)
		}
		go handleConnection(conn)
	}
}

func unpackMsg(packedMsg []byte) *core.Msg {
	var networkBuf bytes.Buffer
	networkBuf.Write(packedMsg)
	msgDec := gob.NewDecoder(&networkBuf)

	var msg core.Msg
	err := msgDec.Decode(&msg)
	if err != nil {
		log.Error("unpackMsgErr", "err", err, "msgBytes", msg)
	}

	return &msg
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		packedMsg, err := reader.ReadBytes('\n')
		if err != nil {
			log.Error("Error reading from connection", "err", err)
		}
		msg := unpackMsg(packedMsg)
		switch msg.MsgType {
		case "ComGetState":
			handleComGetState(msg.Data)
		case "ShardSendState":
			handleShardSendState(msg.Data)
		default:
			log.Error("Unknown message type received", "msgType", msg.MsgType)
		}
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

	committee_ref.HandleShardSendState(&data)
}
