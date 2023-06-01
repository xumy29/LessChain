package core

import (
	"time"
)

// Config is the configuration parameters of mining.
type CommitteeConfig struct {
	Recommit             time.Duration // The time interval for miner to re-create mining work.
	MaxBlockSize         int
	InjectSpeed          int
	Height2Reconfig      int
	MultiSignRequiredNum int
}
