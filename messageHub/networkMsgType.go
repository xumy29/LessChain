package messageHub

const (
	ShardSendGenesis   string = "ShardSendGenesis"
	BooterSendContract string = "BooterSendContract"

	ComGetHeight string = "ComGetHeight"

	ComGetState    string = "ComGetState"
	ShardSendState string = "ShardSendState"

	ClientSendTx        string = "ClientSendTx"
	ClientSetInjectDone string = "ClientSetInjectDone"
	ComSendTxReceipt    string = "ComSendTxReceipt"

	ComSendBlock string = "ComSendBlock"

	LeaderInitMultiSign string = "LeaderInitMultiSign"
	MultiSignReply      string = "MultiSignReply"

	// pbft part
	CPrePrepare        string = "CPrePrepare"
	CPrepare           string = "CPrepare"
	CCommit            string = "CCommit"
	CReply             string = "CReply"
	CRequestOldrequest string = "CRequestOldrequest"
	CSendOldrequest    string = "CSendOldrequest"

	NodeSendInfo string = "NodeSendInfo"
)
