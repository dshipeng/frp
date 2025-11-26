// TCPMux 服务端实现
package tcpmux

import (
	"fmt"
	"log"
	"net"
	"sync"
)

// Server TCPMux 服务器
type Server struct {
	listener     net.Listener
	factory      MuxFactory
	sessions     map[*Session]bool
	sessionsMu   sync.RWMutex
	streamHandler func(net.Conn)
	closed       bool
	closeMu      sync.Mutex
}

// Session 包装的会话
type Session struct {
	conn    net.Conn
	session MuxSession
	factory MuxFactory
}

// RemoteAddr 返回远程地址
func (s *Session) RemoteAddr() net.Addr {
	if s.conn != nil {
		return s.conn.RemoteAddr()
	}
	return nil
}

// LocalAddr 返回本地地址
func (s *Session) LocalAddr() net.Addr {
	if s.conn != nil {
		return s.conn.LocalAddr()
	}
	return nil
}

// NewServer 创建新的服务器
func NewServer(addr string, muxType MuxType) (*Server, error) {
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

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Server{
		listener:      listener,
		factory:       factory,
		sessions:      make(map[*Session]bool),
		streamHandler: defaultStreamHandler,
	}, nil
}

// SetStreamHandler 设置流处理函数
func (s *Server) SetStreamHandler(handler func(net.Conn)) {
	s.streamHandler = handler
}

// Serve 开始服务
func (s *Server) Serve() error {
	log.Printf("TCPMux 服务器启动，监听地址: %s，协议: %s", s.listener.Addr(), s.factory.Type())

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.closeMu.Lock()
			closed := s.closed
			s.closeMu.Unlock()
			if closed {
				return nil
			}
			return err
		}

		// 为每个连接创建会话
		session, err := s.factory.NewServerSession(conn)
		if err != nil {
			log.Printf("创建会话失败: %v", err)
			conn.Close()
			continue
		}

		wrappedSession := &Session{
			conn:    conn,
			session: session,
			factory: s.factory,
		}

		s.sessionsMu.Lock()
		s.sessions[wrappedSession] = true
		s.sessionsMu.Unlock()

		// 处理会话
		go s.handleSession(wrappedSession)
	}
}

// handleSession 处理会话
func (s *Server) handleSession(session *Session) {
	defer func() {
		session.session.Close()
		session.conn.Close()
		s.sessionsMu.Lock()
		delete(s.sessions, session)
		s.sessionsMu.Unlock()
	}()

	log.Printf("新会话建立: %s -> %s (协议: %s)",
		session.conn.RemoteAddr(), session.conn.LocalAddr(), session.factory.Type())

	// 接受并处理新流
	for !session.session.IsClosed() {
		stream, err := session.session.AcceptStream()
		if err != nil {
			if !session.session.IsClosed() {
				log.Printf("接受流失败: %v", err)
			}
			return
		}

		// 处理流
		go s.streamHandler(stream)
	}

	log.Printf("会话关闭: %s", session.conn.RemoteAddr())
}

// GetSessions 获取所有活跃的会话（用于服务端主动打开流）
func (s *Server) GetSessions() []*Session {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	sessions := make([]*Session, 0, len(s.sessions))
	for session := range s.sessions {
		// 只返回未关闭的会话
		if !session.session.IsClosed() {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// OpenStreamOnSession 在指定会话上打开新流（服务端主动发起）
func (s *Server) OpenStreamOnSession(session *Session) (net.Conn, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	s.sessionsMu.RLock()
	exists := s.sessions[session]
	s.sessionsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if session.session.IsClosed() {
		return nil, fmt.Errorf("session is closed")
	}

	return session.session.OpenStream()
}

// GetFirstAvailableSession 获取第一个可用的会话（用于代理服务）
func (s *Server) GetFirstAvailableSession() *Session {
	sessions := s.GetSessions()
	if len(sessions) == 0 {
		return nil
	}
	return sessions[0]
}

// Close 关闭服务器
func (s *Server) Close() error {
	s.closeMu.Lock()
	s.closed = true
	s.closeMu.Unlock()

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return err
		}
	}

	// 关闭所有会话
	s.sessionsMu.Lock()
	for session := range s.sessions {
		session.session.Close()
		session.conn.Close()
	}
	s.sessions = make(map[*Session]bool)
	s.sessionsMu.Unlock()

	return nil
}

// defaultStreamHandler 默认流处理函数（Echo 服务器）
func defaultStreamHandler(stream net.Conn) {
	defer stream.Close()

	log.Printf("新流创建: %s -> %s", stream.RemoteAddr(), stream.LocalAddr())

	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("读取流数据失败: %v", err)
			}
			break
		}

		if n > 0 {
			log.Printf("收到数据 [%s]: %s", stream.RemoteAddr(), string(buf[:n]))

			// Echo 回数据
			if _, err := stream.Write(buf[:n]); err != nil {
				log.Printf("写入流数据失败: %v", err)
				break
			}
		}
	}

	log.Printf("流关闭: %s", stream.RemoteAddr())
}
