package core

const (
	MsgTypeClientInjectTX2Committee uint32 = iota
	MsgTypeCommitteeReply2Client
	MsgTypeSetInjectDone2Committee
	MsgTypeCommitteeAddTB
	MsgTypeCommitteeInitialAddrs
	MsgTypeCommitteeAdjustAddrs
	MsgTypeGetTB

	MsgTypeComGetStateFromShard
	MsgTypeShardSendStateToCom

	MsgTypeAddBlock2Shard
	MsgTypeReady4Reconfig
	MsgTypeTBChainPushTB2Clients
	MsgTypeTBChainPushTB2Coms
)
