package cfg

import "fmt"

var (
	BooterAddr  string
	ClientTable map[uint32]string
	NodeTable   map[uint32]map[int]string
)

// 包被引用时自动执行init函数
func init() {
	BooterAddr = "127.0.0.1:" + fmt.Sprint(18000)

	// 配置1个客户端，但为了可扩展，也实现成map的形式
	ClientTable = make(map[uint32]string)
	ClientTable[0] = "127.0.0.1:" + fmt.Sprint(19000)

	// 配置128个分片，每个分片20个节点的地址
	NodeTable = make(map[uint32]map[int]string)
	currentPort := 20000
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
