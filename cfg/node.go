package cfg

import "fmt"

var (
	GethIPAddr   string
	BooterAddr   string
	ClientTable  map[uint32]string
	NodeTable    map[uint32]map[uint32]string // 分片->节点
	ComNodeTable map[uint32]map[uint32]string // 委员会->节点
)

// 包被引用时自动执行init函数
func init() {
	GethIPAddr = "192.168.3.2"
	BooterAddr = "192.168.3.2:" + fmt.Sprint(18000)

	// 配置1个客户端，但为了可扩展，也实现成map的形式
	ClientTable = make(map[uint32]string)
	ClientTable[0] = "192.168.3.2:" + fmt.Sprint(19000)

	// 配置每个分片20个节点的地址
	NodeTable = make(map[uint32]map[uint32]string)

	startIP := 4
	var i, j uint32
	for i = 0; i < 9; i++ {
		NodeTable[i] = make(map[uint32]string)
		// 从50002开始配置分片节点，每台服务器配置1个分片的节点，端口都是 20000 ~ 20019
		host := fmt.Sprintf("192.168.3.%d", startIP+int(i))
		startPort := 20000
		for j = 0; j < 20; j++ {
			NodeTable[i][j] = host + ":" + fmt.Sprint(startPort)
			startPort += 1
		}
	}

	// 初始时ComNodeTable与NodeTable相等，重组时会发现变化
	ComNodeTable = NodeTable

}
