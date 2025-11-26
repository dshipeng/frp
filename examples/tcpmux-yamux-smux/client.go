// TCPMux 客户端实现
package tcpmux

import (
	"fmt"
	"net"
)

// Client TCPMux 客户端
type Client struct {
	conn    net.Conn
	session MuxSession
	factory MuxFactory
}

// Dial 连接到服务器并创建会话
func Dial(addr string, muxType MuxType) (*Client, error) {
	if !muxType.IsValid() {
		return nil, fmt.Errorf("invalid mux type: %s", muxType)
	}

	var factory MuxFactory
	switch muxType {
	case MuxTypeYamux:
		factory = NewYamuxFactory()
	case MuxTypeSmux:
		factory = NewSmuxFactory()
	default:
		return nil, fmt.Errorf("unsupported mux type: %s", muxType)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	session, err := factory.NewClientSession(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Client{
		conn:    conn,
		session: session,
		factory: factory,
	}, nil
}

// OpenStream 打开新流
func (c *Client) OpenStream() (net.Conn, error) {
	if c.session.IsClosed() {
		return nil, fmt.Errorf("session closed")
	}
	return c.session.OpenStream()
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.session != nil {
		c.session.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Session 返回底层会话
func (c *Client) Session() MuxSession {
	return c.session
}

// Factory 返回使用的工厂
func (c *Client) Factory() MuxFactory {
	return c.factory
}
