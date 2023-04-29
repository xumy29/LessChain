package core

type MessageHub interface {
	// Send(toid int, msgType uint64, msg interface{})
	Send(msgType uint64, toid uint64, msg interface{}, callback func(interface{}))
}
