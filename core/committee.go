package core

type Committee interface {
	Start(nodeId int)
	Close()

	SetInjectTXDone(uint32)
	CanStopV1() bool
	CanStopV2() bool

	NewBlockGenerated(block *Block)

	StartWorker()

	GetMembers() []*NodeInfo
	AddMember(*NodeInfo)
}
