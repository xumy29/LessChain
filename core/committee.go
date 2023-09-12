package core

type Committee interface {
	Start(nodeId int)
	Close()

	SetInjectTXDone()
	CanStopV1() bool
	CanStopV2() bool

	NewBlockGenerated(block *Block)
}
