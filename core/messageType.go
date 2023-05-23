package core

const (
	MsgTypeShardReply2Client uint64 = iota
	MsgTypeClientInjectTX2Committee
	MsgTypeSetInjectDone2Shard
	MsgTypeAddTB
	MsgTypeGetTB
	MsgTypeComGetState
	MsgTypeAddBlock2Shard
	MsgTypeReady4Reconfig
)
