package core

const (
	MsgTypeItxReply uint64 = iota
	MsgTypeCtx1Reply
	MsgTypeCtx2Reply
	MsgTypeShardReply2Client
	MsgTypeClientInjectTX2Shard
	MsgTypeSetInjectDone2Shard
)
