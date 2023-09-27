package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser // 由构建函数传入
	buf  *bufio.Writer      // 为了防阻塞而创建的带缓冲的 Writer
	dec  *gob.Decoder       // 解码器
	enc  *gob.Encoder       // 编码器
}

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

func (c *GobCodec) ReadHeader(h *Header) error {
	// 解析 conn 里的内容到 h 中
	return c.dec.Decode(h)
}

func (c *GobCodec) ReadBody(body interface{}) error {
	// 解析 conn 里的内容到 body 中
	return c.dec.Decode(body)
}

func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	// 编码 h 内容写入 c.buf 中
	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	// 编码 body 内容写入 c.buf 中
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}
	return nil
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}

// 将 nil 转化为 *GobCodec 类型，赋值后丢弃
// 本质是验证 GobCodec 有没有实现 Codec 接口的所有方法
var _ Codec = (*GobCodec)(nil)
