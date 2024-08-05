package singleflight

import "sync"

// 代表正在进行中，或已经结束的请求
type call struct {
	wg  sync.WaitGroup //避免重入
	val interface{}
	err error
}

//管理不同 key 的请求(call)
type Group struct {
	mu sync.Mutex       // 保护 Group 的成员变量 m 不被并发读写而加上的锁
	m  map[string]*call // 延迟初始化
}

//相同的 key，无论Do被调用几次，函数fn只调用一次，调用结束以后返回 返回值或错误。
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call) // 延迟初始化
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()           // 如果请求正在进行中，则阻塞，直到锁被释放。
		return c.val, c.err   // 请求结束，返回结果
	}
	c := new(call)
	c.wg.Add(1)                // 发起请求前加锁，锁加1
	g.m[key] = c               // 添加到 g.m，表明 key 已经有对应的请求在处理
	g.mu.Unlock()

	c.val, c.err = fn()        // 调用 fn，发起请求
	c.wg.Done()                // 请求结束，锁减1

	g.mu.Lock()
	delete(g.m, key)            // 更新 g.m
	g.mu.Unlock()

	return c.val, c.err         // 返回结果
}
