package core

type MessageHub interface {
	// Send(toid int, msgType uint64, msg interface{})
	Send(msgType uint64, toid int, msg interface{})
}
