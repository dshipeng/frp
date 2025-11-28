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
	listener net.Listener
	sessions map[*Session]bool
	sessionsMu sync.RWMutex
	streamHandler func(*Stream)
	closed bool
	closeMu sync.Mutex
}

// NewServer 创建新的服务器
func NewServer(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Server{
		listener:      listener,
		sessions:      make(map[*Session]bool),
		streamHandler: defaultStreamHandler,
	}, nil
}

// SetStreamHandler 设置流处理函数
func (s *Server) SetStreamHandler(handler func(*Stream)) {
	s.streamHandler = handler
}

// Serve 开始服务
func (s *Server) Serve() error {
	log.Printf("TCPMux 服务器启动，监听地址: %s", s.listener.Addr())

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
		session := NewSession(conn, true)
		s.sessionsMu.Lock()
		s.sessions[session] = true
		s.sessionsMu.Unlock()

		// 处理会话
		go s.handleSession(session)
	}
}

// handleSession 处理会话
func (s *Server) handleSession(session *Session) {
	defer func() {
		session.Close()
		s.sessionsMu.Lock()
		delete(s.sessions, session)
		s.sessionsMu.Unlock()
	}()

	// 接受并处理新流
	for !session.IsClosed() {
		stream, err := session.AcceptStream()
		if err != nil {
			return
		}

		// 处理流
		go s.streamHandler(stream)
	}
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
		session.Close()
	}
	s.sessions = make(map[*Session]bool)
	s.sessionsMu.Unlock()

	return nil
}

// defaultStreamHandler 默认流处理函数
func defaultStreamHandler(stream *Stream) {
	log.Printf("新流创建: StreamID=%d", stream.ID())
	
	// 简单的 echo 服务器
	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("读取流数据失败: %v", err)
			}
			break
		}

		log.Printf("收到数据 [StreamID=%d]: %s", stream.ID(), string(buf[:n]))
		
		// Echo 回数据
		if _, err := stream.Write(buf[:n]); err != nil {
			log.Printf("写入流数据失败: %v", err)
			break
		}
	}

	stream.Close()
	log.Printf("流关闭: StreamID=%d", stream.ID())
}

