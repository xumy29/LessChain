package core

const (
	MsgTypeClientInjectTX2Committee uint64 = iota
	MsgTypeCommitteeReply2Client
	MsgTypeSetInjectDone2Committee
	MsgTypeCommitteeAddTB
	MsgTypeGetTB
	MsgTypeComGetStateFromShard
	MsgTypeAddBlock2Shard
	MsgTypeReady4Reconfig
	MsgTypeTBChainPushTB2Clients
)
