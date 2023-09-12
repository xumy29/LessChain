package core

type MessageHub interface {
	Send(msgType uint32, id uint32, msg interface{}, callback func(...interface{}))
}
