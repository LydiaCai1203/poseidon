package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"poseidon/codec"
	"poseidon/poseidon"
	"time"
)

func startServer(addr chan string) {
	// :0 表示让操作系统随机分配一个可用端口号
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	poseidon.Accept(l)
}

func main() {
	addr := make(chan string)
	go startServer(addr)

	// 建立连接
	conn, _ := net.Dial("tcp", <-addr)
	defer func() {
		_ = conn.Close()
	}()

	time.Sleep(time.Second)
	// 写了一个 option
	// 将 DefaultOption 对象编码成 JSON 传到 conn 里去
	_ = json.NewEncoder(conn).Encode(poseidon.DefaultOption)
	cc := codec.NewGobCodec(conn)
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		// 写了一个 header 和 body
		_ = cc.Write(h, fmt.Sprintf("rpc %d", h.Seq))
		// 读取响应
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}
