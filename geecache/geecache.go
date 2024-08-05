package geecache

import "fmt"
import "log"
import "sync"
import "geecache/singleflight"
import pb "geecache/geecachepb"

type Group struct{
	name string
	getter Getter  //缓存未命中时获取源数据的回调(callback)
	mainCache cache
	peers     PeerPicker
	loader *singleflight.Group //使用singleflight Group来确保每个键只被获取一次
}

//接口 Getter 和 回调函数 Get(key string)([]byte, error)
type Getter interface {
	Get(key string) ([]byte, error)
}

//接口型函数--函数类型实现某一个接口
type GetterFunc func (key string)  ([]byte,error)

func (f GetterFunc) Get(key string) ([]byte,error){
	return f(key)
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

//实例化 Group
func NewGroup(name string,cacheBytes int64, getter Getter) *Group{
	if getter==nil{
		panic("nil getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g:=&Group{
		name:name,
		getter:getter,
		mainCache:cache{cacheBytes:cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name]=g //将 group 存储在全局变量 groups 中
	return g
}

func GetGroup(name string) *Group{
	mu.RLock()
	defer mu.RUnlock()
	g:=groups[name]
	return g	
}

func (g *Group) Get(key string) (ByteView,error){
	if key==""{
		return ByteView{},fmt.Errorf("key is required")
	}

	//从 mainCache 中查找缓存，如果存在则返回缓存值
	if v,ok:=g.mainCache.get(key);ok{
		log.Println("[GeeCache] hit")
		return v,nil
	}

	//缓存不存在，则调用 load 方法
	return g.load(key)
}

//将实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

//load 调用 getLocally
func (g *Group) load(key string) (value ByteView,err error){
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			//使用 PickPeer() 方法选择节点
			if peer, ok := g.peers.PickPeer(key); ok {
				//非本机节点，则调用 getFromPeer() 从远程获取
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}

		//本机节点或失败，则回退到 getLocally()
		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

//getLocally 调用用户回调函数 g.getter.Get() 获取源数据
func (g *Group) getLocally(key string) (ByteView,error){
	bytes,err:=g.getter.Get(key)
	if err!=nil{
		return ByteView{},err
	}

	value:=ByteView{b:cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

//将源数据添加到缓存 mainCache 中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}


//使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req:=&pb.Request{
		Group:g.name,
		Key:key,
	}
	res:=&pb.Response{}

	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}