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
	MsgTypeSetInjectDone2Nodes
	MsgTypeComAddTb2TBChain
	MsgTypeCommitteeInitialAddrs
	MsgTypeCommitteeAdjustAddrs
	MsgTypeGetTB

	MsgTypeSendBlock2Shard
	MsgTypeReady4Reconfig
	MsgTypeTBChainPushTB2Client
	MsgTypeTBChainPushTB2Coms

	MsgTypeComGetLatestBlockHashFromEthChain
)
