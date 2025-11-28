// TCPMux 客户端实现
package tcpmux

import (
	"fmt"
	"net"
)

// Client TCPMux 客户端
type Client struct {
	session *Session
}

// Dial 连接到服务器并创建会话
func Dial(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	session := NewSession(conn, false)
	return &Client{session: session}, nil
}

// OpenStream 打开新流
func (c *Client) OpenStream() (*Stream, error) {
	if c.session.IsClosed() {
		return nil, fmt.Errorf("session closed")
	}
	return c.session.OpenStream()
}

// Close 关闭客户端
func (c *Client) Close() error {
	return c.session.Close()
}

// Session 返回底层会话
func (c *Client) Session() *Session {
	return c.session
}
