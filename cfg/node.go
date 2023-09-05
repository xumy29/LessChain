package cfg

import "fmt"

var (
	ClientAddr string
	NodeTable  map[uint32]map[int]string
)

// 包被引用时自动执行init函数
func init() {
	NodeTable = make(map[uint32]map[int]string)

	currentPort := 19999
	ClientAddr = "127.0.0.1:" + fmt.Sprint(currentPort)
	currentPort += 1

	// 配置128个分片，每个分片20个节点的地址
	var i uint32
	for i = 0; i < 128; i++ {
		NodeTable[i] = make(map[int]string)
		host := "127.0.0.1"
		for j := 0; j < 20; j++ {
			NodeTable[i][j] = host + ":" + fmt.Sprint(currentPort)
			currentPort += 1
		}
	}
}
