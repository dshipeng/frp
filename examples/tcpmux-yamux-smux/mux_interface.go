// 多路复用接口定义
package tcpmux

import (
	"net"
)

// MuxSession 表示一个多路复用会话
type MuxSession interface {
	// OpenStream 打开一个新的流（客户端）
	OpenStream() (net.Conn, error)
	// AcceptStream 接受一个新的流（服务端）
	AcceptStream() (net.Conn, error)
	// Close 关闭会话
	Close() error
	// IsClosed 检查会话是否已关闭
	IsClosed() bool
	// NumStreams 返回当前流的数量
	NumStreams() int
}

// MuxFactory 用于创建多路复用会话的工厂接口
type MuxFactory interface {
	// NewClientSession 创建客户端会话
	NewClientSession(conn net.Conn) (MuxSession, error)
	// NewServerSession 创建服务端会话
	NewServerSession(conn net.Conn) (MuxSession, error)
	// Type 返回协议类型
	Type() MuxType
}
