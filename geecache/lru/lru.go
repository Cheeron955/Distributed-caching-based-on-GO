package lru

import "container/list"

type Cache struct {
	maxBytes  int64                         //允许使用的最大内存
	nbytes    int64                         //当前已使用的内存
	ll        *list.List                    // Go 语言标准库实现的双向链表
	cache     map[string]*list.Element      //键是字符串，值是双向链表中对应节点的指针。
	OnEvicted func(key string, value Value) //某条记录被移除时的回调函数
}

// 双向链表节点的数据类型
// 链表中保存key，可以在移除节点的时候，使用key方便的移除
type entry struct {
	key   string
	value Value
}

// 返回值所占用的内存大小
type Value interface {
	Len() int
}

// Cache的构造函数,方便实例化
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 查找 从字典中找到对应的双向链表的节点并将该节点移动到队尾（约定 front 为队尾）。
func (c *Cache) Get(key string) (value Value, ok bool) {
	//从map中查找key对应的值，如果key不存在ok==false
	if ele, ok := c.cache[key]; ok == true {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry) //(*entry)表示将Value转成*entry类型访问
		return kv.value, true
	}
	return
}

// 删除 移除最近最少访问的节点（队首）
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key) //参数为 map 和其对应的 key
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 新增/修改
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		//键不存在，队尾添加新节点
		ele := c.ll.PushFront(&entry{key, value})
		//字典中添加 key 和节点的映射关系
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}

	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
