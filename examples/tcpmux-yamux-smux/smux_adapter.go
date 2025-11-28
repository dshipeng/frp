// smux 适配器实现
package tcpmux

import (
	"io"
	"net"
	"sync"
	"sync/atomic"

	smux "github.com/xtaci/smux"
)

// smuxSession smux 会话包装
type smuxSession struct {
	session *smux.Session
	closed  int32
	mu      sync.RWMutex
}

// smuxFactory smux 工厂
type smuxFactory struct{}

// NewSmuxFactory 创建 smux 工厂
func NewSmuxFactory() MuxFactory {
	return &smuxFactory{}
}

// Type 返回协议类型
func (f *smuxFactory) Type() MuxType {
	return MuxTypeSmux
}

// NewClientSession 创建客户端会话
func (f *smuxFactory) NewClientSession(conn net.Conn) (MuxSession, error) {
	config := smux.DefaultConfig()
	config.KeepAliveInterval = 30
	config.KeepAliveTimeout = 90

	session, err := smux.Client(conn, config)
	if err != nil {
		return nil, err
	}

	return &smuxSession{
		session: session,
		closed:  0,
	}, nil
}

// NewServerSession 创建服务端会话
func (f *smuxFactory) NewServerSession(conn net.Conn) (MuxSession, error) {
	config := smux.DefaultConfig()
	config.KeepAliveInterval = 30
	config.KeepAliveTimeout = 90

	session, err := smux.Server(conn, config)
	if err != nil {
		return nil, err
	}

	return &smuxSession{
		session: session,
		closed:  0,
	}, nil
}

// OpenStream 打开新流
func (s *smuxSession) OpenStream() (net.Conn, error) {
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
func (s *smuxSession) AcceptStream() (net.Conn, error) {
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
func (s *smuxSession) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil
	}

	return s.session.Close()
}

// IsClosed 检查是否已关闭
func (s *smuxSession) IsClosed() bool {
	return atomic.LoadInt32(&s.closed) == 1 || s.session.IsClosed()
}

// NumStreams 返回流的数量（smux 不直接提供，返回 -1 表示未知）
func (s *smuxSession) NumStreams() int {
	// smux 不提供直接获取流数量的方法
	return -1
}
