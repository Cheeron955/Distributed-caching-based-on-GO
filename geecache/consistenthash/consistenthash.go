package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	hash     Hash           //Hash 函数,默认为 crc32.ChecksumIEEE 算法
	replicas int            //虚拟节点倍数
	keys     []int          // 哈希环
	hashMap  map[int]string //虚拟节点与真实节点的映射表-键是虚拟节点的哈希值，值是真实节点的名称。
}

// 构造函数 New() 允许自定义虚拟节点倍数和 Hash 函数
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}

	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}

	return m
}

// 添加真实节点/机器的方法,允许传入 0 或 多个真实节点的名称
func (m *Map) Add(keys ...string) {
	//真实节点key
	for _, key := range keys {
		//创建 m.replicas 个虚拟节点
		for i := 0; i < m.replicas; i++ {
			//虚拟节点的名称,并使用m.hash()计算虚拟节点的哈希值
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash) //添加到环上
			m.hashMap[hash] = key         //增加虚拟节点和真实节点的映射关系
		}
	}
	sort.Ints(m.keys) //环上的哈希值排序 Ints 类型
}

func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	//计算 key 的哈希值
	hash := int(m.hash([]byte(key)))
	// 顺时针找到第一个匹配的虚拟节点的下标
	// 使用二分查找，[0,len(m.keys))中找到一个idx
	// 使函数func(idx)为True的最小值，并且func(idx+1)也为True。
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]]
}
