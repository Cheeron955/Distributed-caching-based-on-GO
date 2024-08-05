package geecache
 
import pb "geecache/geecachepb"

type PeerPicker interface{
	//用于根据传入的 key 选择相应节点 PeerGetter
	PickPeer(key string) (peer PeerGetter,ok bool)
}

type PeerGetter interface{
	//从对应 group 查找缓存值
	Get(in *pb.Request, out *pb.Response) error
}
