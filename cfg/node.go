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

	// 从50002开始配置分片节点
	// 配置每个分片20个节点的地址
	NodeTable = make(map[uint32]map[uint32]string)

	hostTable := make(map[uint32]string)
	startPortTable := make(map[uint32]uint32)
	hostTable[0] = "192.168.3.3"
	startPortTable[0] = 20000
	hostTable[1] = "192.168.3.3"
	startPortTable[1] = 20020
	hostTable[2] = "192.168.3.4"
	startPortTable[2] = 20000
	hostTable[3] = "192.168.3.4"
	startPortTable[3] = 20020
	hostTable[4] = "192.168.3.5"
	startPortTable[4] = 20000
	hostTable[5] = "192.168.3.5"
	startPortTable[5] = 20020
	hostTable[6] = "192.168.3.6"
	startPortTable[6] = 20000
	hostTable[7] = "192.168.3.6"
	startPortTable[7] = 20020
 
  hostTable[8] = "192.168.3.7"
  startPortTable[8] = 20000
	hostTable[9] = "192.168.3.7"
	startPortTable[9] = 20020
	hostTable[10] = "192.168.3.8"
	startPortTable[10] = 20000
	hostTable[11] = "192.168.3.8"
	startPortTable[11] = 20020
	hostTable[12] = "192.168.3.9"
	startPortTable[12] = 20000
	hostTable[13] = "192.168.3.9"
	startPortTable[13] = 20020
	hostTable[14] = "192.168.3.10"
	startPortTable[14] = 20000
	hostTable[15] = "192.168.3.10"
	startPortTable[15] = 20020


	var i, j uint32
	for i = 0; i < uint32(len(hostTable)); i++ {
		NodeTable[i] = make(map[uint32]string)
		for j = 0; j < 20; j++ {
			NodeTable[i][j] = hostTable[i] + ":" + fmt.Sprint(startPortTable[i]+j)
		}
	}

	// 初始时ComNodeTable与NodeTable相等，重组时会发现变化
	ComNodeTable = NodeTable

}
