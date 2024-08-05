package geecache

// 只读数据结构 ByteView 用来表示缓存值
type ByteView struct {
	b []byte //存储真实的缓存值, byte 类型--能够支持任意的数据类型的存储，例如字符串、图片等
}

// 返回其所占的内存大小
func (v ByteView) Len() int {
	return len(v.b)
}

// 返回一个拷贝，防止缓存值被外部程序修改
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// 以字符串形式返回数据，必要时生成副本
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
