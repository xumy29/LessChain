package messageHub

import (
	"go-w3chain/core"
	"math/rand"
)

func toReconfig() {
	isNotToReconfig -= 1
	if isNotToReconfig == 0 {
		// 这里需要开一个新线程，不要与最后一个调用此函数的委员会共用一个线程
		go reconfig()
		isNotToReconfig = len(committees_ref)
	}
}

/* 打乱allnodes_id，重新设定委员会中的节点 */
func reconfig() {
	// 设置随机种子
	rand.Seed(int64(getSeed()))

	rand.Shuffle(len(allnodes_id), func(i, j int) {
		allnodes_id[i], allnodes_id[j] = allnodes_id[j], allnodes_id[i]
	})

	begin := 0
	for _, com := range committees_ref {
		node_num := len(com.Nodes)
		nodes_index := allnodes_id[begin : begin+node_num]
		begin += node_num

		nodes_4_com := make([]*core.Node, node_num)
		for j, index := range nodes_index {
			nodes_4_com[j] = nodes_ref[index]
		}

		com.SetReconfigRes(nodes_4_com)
	}
}

/* 从信标链获取最新高度的已确认区块（比如有足够数量的区块跟在后面）的信息，返回一个整数作为重组的种子 */
func getSeed() int {
	seed := rand.Int()
	// TODO: implement me
	return seed
}
