package shard

import (
	"bufio"
	"encoding/json"
	"go-w3chain/core"
	"go-w3chain/log"
	"net"
	"time"
)

//节点使用的tcp监听
func (shard *Shard) TcpListen() {
	listen, err := net.Listen("tcp", shard.leader.GetAddr())
	if err != nil {
		log.Error("Node tcp listien err:" + err.Error())
	}
	log.Debug("Node begin TCP listen", " addr=", shard.leader.GetAddr())
	defer listen.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Warn("Node tcp accept err:" + err.Error())
		}
		go shard.handleConn(conn)
		time.Sleep(10 * time.Millisecond)
	}
}

func (shard *Shard) handleConn(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		// b, err := ioutil.ReadAll(conn)
		b, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			log.Warn("Node read data err:" + err.Error())
		} else {
			if len(b) > 0 {
				shard.handleMessage([]byte(b))
			}
		}
		// time.Sleep(1 * time.Millisecond)
	}
}

func (shard *Shard) handleMessage(data []byte) {
	//切割消息，根据消息命令调用不同的功能
	cmd, content := splitMessage(data)
	switch command(cmd) {
	// TODO: 根据不同消息进行处理
	case cHeader:
		shard.handleHeader(content)
	default:
		log.Error("Unkown message.", "cmd", cmd)
	}
}

func (shard *Shard) handleHeader(content []byte) {
	header := new(core.Header)
	err := json.Unmarshal(content, header)
	if err != nil {
		log.Error("Unmarshal header err:"+err.Error(), "content", string(content))
	}
	log.Trace("Unmarshal header successed!")
	// todo: 存储到 db
	// str := fmt.Sprintf("Shard %d has received the [header.Number=%d] from shard %d.\n", shard.leader.chainID, header.Number, header.ShardID)
	// log.Info(str)
}

//使用tcp发送消息
func (shard *Shard) TcpDial(context []byte, addr string) {
	/* 给 map 访问加读写锁 */
	shard.connMaplock.Lock()
	if _, ok := shard.connMap[addr]; !ok {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Error("connect error:"+err.Error(), "addr", addr,
				"context", context, "cur shardid", shard.shardID)
			return
		}
		shard.connMap[addr] = conn
	}
	conn := shard.connMap[addr]
	/* 给 map 解锁 */
	shard.connMaplock.Unlock()

	// _, err := conn.Write(context)
	msgstr := string(context) + "\n"
	msgb := []byte(msgstr)
	_, err := conn.Write(msgb)
	if err != nil {
		log.Error("write error:"+err.Error(), "addr", addr,
			"context", context, "cur shardid", shard.shardID)
	}
}

func (shard *Shard) StopTCPConn() {
	shard.connMaplock.Lock()
	for addr, conn := range shard.connMap {
		conn.Close()
		log.Trace("close TCP connection..", "closed addr", addr, "cur shardid", shard.shardID)
	}
	shard.connMaplock.Unlock()
}
