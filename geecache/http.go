package geecache

import (
	"fmt"
	"geecache/consistenthash"
	pb "geecache/geecachepb"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
)

const defaultBasePath = "/_geecache/"
const defaultReplicas = 50

// HTTPPool为HTTP对等体池实现了PeerPicker。
type HTTPPool struct {
	self        string                 //记录自己的地址--主机名/IP 和端口 ---https://example.net:8000
	basePath    string                 //节点间通讯地址的前缀,默认是 /_geecache/
	mu          sync.Mutex             // guards peers and httpGetters
	peers       *consistenthash.Map    //根据具体的 key 选择节点
	httpGetters map[string]*httpGetter // 映射远程节点与对应的 httpGetter
}

// NewHTTPPool初始化对等体的HTTP池。
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// 带有服务器名称的日志信息
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	p.Log("%s %s", r.Method, r.URL.Path)

	//约定访问路径格式为 /<basepath>/<groupname>/<key>
	//r.URL.Path[len(p.basePath):]---><groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2) //[<groupname> <key>]

	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//将值作为原型消息写入响应体。
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//字节流，浏览器处理字节流的默认方式就是下载
	w.Header().Set("Content-Type", "application/octet-stream")

	//将缓存值作为 httpResponse 的 body 返回
	w.Write(body)

}

type httpGetter struct {
	baseURL string //将要访问的远程节点的地址,例如 http://example.com/_geecache/
}

// 获取返回值，并转换为 []bytes 类型
func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()), //.pb.go中的函数
		url.QueryEscape(in.GetKey()),
		//QueryEscape 对字符串进行转义，以便可以将其安全地放置在 URL 查询中。
	)

	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	//ioutil.ReadAll读取整个文件到内存
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}
	//反序列化
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

var _ PeerGetter = (*httpGetter)(nil)

// 实例化了一致性哈希算法，并且添加了传入的节点
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		//每一个节点创建了一个 HTTP 客户端 httpGetter
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

var _ PeerPicker = (*HTTPPool)(nil)

// 包装了一致性哈希算法的 Get() 方法
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		//具体的 key，选择节点，返回节点对应的 HTTP 客户端。
		return p.httpGetters[peer], true
	}
	return nil, false
}
