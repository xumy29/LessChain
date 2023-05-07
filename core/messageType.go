package core

const (
	MsgTypeShardReply2Client uint64 = iota
	MsgTypeClientInjectTX2Shard
	MsgTypeSetInjectDone2Shard
	MsgTypeAddTB
	MsgTypeGetTB
	MsgTypeComGetTX
	MsgTypeAddBlock2Shard
	MsgTypeReady4Reconfig
)
