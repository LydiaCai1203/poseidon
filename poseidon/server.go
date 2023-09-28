/*
设计协议协商的这部分信息，需要设计固定的字节来传输
第 1 个字节: 表示序列化的方式
第 2 个字节: 表示压缩方式
第 3-6 个字节: 表示 header 的长度
第 7-10 个字节: 表示 body 的长度

1. 请求格式
Option{MagicNumber: xxx, CodecType: xxx}
Header{ServiceMethod...}
Body interface{}
option | header | body | header | body | ...

2. 启动服务
lis, _ := net.Listen("tcp", ":999")
Accept(lis)
*/
package poseidon

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"poseidon/codec"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int
	CodecType   codec.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func (server *Server) Accept(listener net.Listener) {
	for {
		// 等待客户端的连接
		conn, err := listener.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		// 处理连接
		go server.ServeConn(conn)
	}
}

func Accept(listener net.Listener) {
	DefaultServer.Accept(listener)
}

// ---------------------------------------------------------------

func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() {
		_ = conn.Close()
	}()
	// 一个连接的开始肯定以 Option 开头
	// 从 conn 中读数据然后写入 opt 对象中
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	// poseidon request 的特殊标记
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	// 开始解码请求
	server.serveCodec(f(conn))
}

var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	// 一次连接允许接收多个请求
	for {
		// 读取请求(1)
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		// 表示有一个协程要开始了
		wg.Add(1)
		go server.handleRequest(cc, req, sending, wg)
	}
	// 等到所有的请求都处理完并回复完
	wg.Wait()
	_ = cc.Close()
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
}

// 读取 cc.conn 里的 header 内容并返回
func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

// 将连接里的数据读取到 request 对象里并返回
func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// 通过反射创建一个新的空白参数，空字符串类型
	req.argv = reflect.New(reflect.TypeOf(""))
	// 从连接中读取并解码到 req.argv 中
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}

// 将 h && body 里的内容写入 cc.buf 里
func (server *Server) sendResponse(
	cc codec.Codec,
	h *codec.Header,
	body interface{},
	sending *sync.Mutex,
) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(
	cc codec.Codec,
	req *request,
	sending *sync.Mutex,
	wg *sync.WaitGroup,
) {
	// 表示该协程已完成
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("rpc resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}
