package cfg

import "fmt"

var (
	BooterAddr   string
	ClientTable  map[uint32]string
	NodeTable    map[uint32]map[uint32]string // 分片->节点
	ComNodeTable map[uint32]map[uint32]string // 委员会->节点
)

// 包被引用时自动执行init函数
func init() {
	BooterAddr = "127.0.0.1:" + fmt.Sprint(18000)

	// 配置1个客户端，但为了可扩展，也实现成map的形式
	ClientTable = make(map[uint32]string)
	ClientTable[0] = "127.0.0.1:" + fmt.Sprint(19000)

	// 配置20个分片，每个分片20个节点的地址
	NodeTable = make(map[uint32]map[uint32]string)
	currentPort := 20000
	var i, j uint32
	for i = 0; i < 20; i++ {
		NodeTable[i] = make(map[uint32]string)
		host := "127.0.0.1"
		for j = 0; j < 20; j++ {
			NodeTable[i][j] = host + ":" + fmt.Sprint(currentPort)
			currentPort += 1
		}
	}

	// 初始时ComNodeTable与NodeTable相等，重组时会发现变化
	ComNodeTable = NodeTable

}
