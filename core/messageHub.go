package core

type MessageHub interface {
	/* messageHub 接口，由于callback必须在调用此接口的函数结束前被调用，所以接口函数必须实现成同步函数 */
	Send(msgType uint64, id uint64, msg interface{}, callback func(...interface{}))
}
