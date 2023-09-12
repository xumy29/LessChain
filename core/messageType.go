package core

const (
	MsgTypeNone uint32 = iota

	MsgTypeComGetHeightFromShard

	MsgTypeComGetStateFromShard
	MsgTypeShardSendStateToCom

	MsgTypeShardSendGenesis
	MsgTypeBooterSendContract

	MsgTypeClientInjectTX2Committee
	MsgTypeCommitteeReply2Client
	MsgTypeSetInjectDone2Committee
	MsgTypeCommitteeAddTB
	MsgTypeCommitteeInitialAddrs
	MsgTypeCommitteeAdjustAddrs
	MsgTypeGetTB

	MsgTypeSendBlock2Shard
	MsgTypeReady4Reconfig
	MsgTypeTBChainPushTB2Clients
	MsgTypeTBChainPushTB2Coms
)
