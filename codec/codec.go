package codec

import "io"

// 一个典型的 RPC 调用如下
// err := client.Call("Mutiply",  args, &reply)
type Header struct {
	ServiceMethod string // 服务名和方法名
	Seq           uint64 // 请求的序号，用于区分请求
	Error         string // 错误信息
}

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

// 抽象出对消息体进行编解码的接口
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewCodecFunc func(io.ReadWriteCloser) Codec

var NewCodecFuncMap map[Type]NewCodecFunc

// 编码类型: 编码方法
func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
