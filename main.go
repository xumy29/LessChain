package main

// GO111MODULE=on go run main.go

import (
	"fmt"

	"go-w3chain/controller"

	"github.com/spf13/pflag"
)

var mode = pflag.StringP("mode", "m", "debug", "mode (run or debug)")
var role = pflag.StringP("role", "r", "booter", "role type (booter, node or client)")
var shardNum = pflag.Int32P("shardNum", "S", 1, "number of shards(and committees)")
var shardId = pflag.Int32P("shardId", "s", 0, "shard id)")

/** go build -o brokerChain.exe
 * brokerChain.exe -m run >> nohup.out 2>&1
 */
func main() {

	pflag.Parse()

	cfgfilename := "cfg/debug.json"
	if *mode == "run" {
		cfgfilename = "cfg/run.json"
	} else if *mode != "debug" {
		fmt.Println("wrong mode")
		return
	}
	fmt.Println("cfg file:", cfgfilename)

	controller.Main(cfgfilename, *role, *shardNum, *shardId)
}
