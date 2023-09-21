package core

type MessageHub interface {
	Send(msgType uint32, id uint32, msg interface{}, callback func(...interface{}))
}

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

	MsgTypeLeaderInitMultiSign
	MsgTypeSendMultiSignReply

	//////////////////////
	//// pbft module /////
	//////////////////////
	MsgTypePbftPropose
	MsgTypePbftPrePrepare
	MsgTypePbftPrepare
	MsgTypePbftCommit
	MsgTypePbftReply
	MsgTypePbftRequestOldMessage
	MsgTypePbftSendOldMessage

	MsgTypeNodeSendInfo2Leader
)
