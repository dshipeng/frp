// yamux 适配器实现
package tcpmux

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	fmux "github.com/hashicorp/yamux"
)

// yamuxSession yamux 会话包装
type yamuxSession struct {
	session *fmux.Session
	closed  int32
	mu      sync.RWMutex
}

// yamuxFactory yamux 工厂
type yamuxFactory struct{}

// NewYamuxFactory 创建 yamux 工厂
func NewYamuxFactory() MuxFactory {
	return &yamuxFactory{}
}

// Type 返回协议类型
func (f *yamuxFactory) Type() MuxType {
	return MuxTypeYamux
}

// NewClientSession 创建客户端会话
func (f *yamuxFactory) NewClientSession(conn net.Conn) (MuxSession, error) {
	config := fmux.DefaultConfig()
	config.EnableKeepAlive = true
	config.KeepAliveInterval = 30 * time.Second

	session, err := fmux.Client(conn, config)
	if err != nil {
		return nil, err
	}

	return &yamuxSession{
		session: session,
		closed:  0,
	}, nil
}

// NewServerSession 创建服务端会话
func (f *yamuxFactory) NewServerSession(conn net.Conn) (MuxSession, error) {
	config := fmux.DefaultConfig()
	config.EnableKeepAlive = true
	config.KeepAliveInterval = 30 * time.Second

	session, err := fmux.Server(conn, config)
	if err != nil {
		return nil, err
	}

	return &yamuxSession{
		session: session,
		closed:  0,
	}, nil
}

// OpenStream 打开新流
func (s *yamuxSession) OpenStream() (net.Conn, error) {
	if s.IsClosed() {
		return nil, io.ErrClosedPipe
	}

	stream, err := s.session.OpenStream()
	if err != nil {
		return nil, err
	}

	return stream, nil
}

// AcceptStream 接受新流
func (s *yamuxSession) AcceptStream() (net.Conn, error) {
	if s.IsClosed() {
		return nil, io.ErrClosedPipe
	}

	stream, err := s.session.AcceptStream()
	if err != nil {
		return nil, err
	}

	return stream, nil
}

// Close 关闭会话
func (s *yamuxSession) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil
	}

	return s.session.Close()
}

// IsClosed 检查是否已关闭
func (s *yamuxSession) IsClosed() bool {
	return atomic.LoadInt32(&s.closed) == 1 || s.session.IsClosed()
}

// NumStreams 返回流的数量（yamux 不直接提供，返回 -1 表示未知）
func (s *yamuxSession) NumStreams() int {
	// yamux 不提供直接获取流数量的方法
	return -1
}
