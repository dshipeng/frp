// Stream 表示一个逻辑流
package tcpmux

import (
	"io"
	"sync"
	"time"
)

// Stream 表示一个多路复用的逻辑流
type Stream struct {
	id       uint32
	session  *Session
	readCh   chan []byte
	writeCh  chan []byte
	closeCh  chan struct{}
	closed   bool
	closeMu  sync.Mutex
	readMu   sync.Mutex
	writeMu  sync.Mutex
}

// Read 从流中读取数据
func (s *Stream) Read(p []byte) (n int, err error) {
	s.readMu.Lock()
	defer s.readMu.Unlock()

	if s.closed {
		return 0, io.EOF
	}

	select {
	case data := <-s.readCh:
		if len(data) == 0 {
			return 0, io.EOF
		}
		n = copy(p, data)
		if n < len(data) {
			// 数据太大，需要缓存剩余部分（简化实现，实际应该用缓冲区）
			go func() {
				s.readCh <- data[n:]
			}()
		}
		return n, nil
	case <-s.closeCh:
		return 0, io.EOF
	case <-time.After(30 * time.Second):
		return 0, io.ErrClosedPipe
	}
}

// Write 向流中写入数据
func (s *Stream) Write(p []byte) (n int, err error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.closed {
		return 0, io.ErrClosedPipe
	}

	// 将数据发送到会话层
	return s.session.writeStream(s.id, p)
}

// Close 关闭流
func (s *Stream) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.closeCh)

	// 发送 FIN 帧
	return s.session.closeStream(s.id)
}

// ID 返回流的 ID
func (s *Stream) ID() uint32 {
	return s.id
}

// LocalAddr 返回本地地址（简化实现）
func (s *Stream) LocalAddr() string {
	return s.session.conn.LocalAddr().String()
}

// RemoteAddr 返回远程地址（简化实现）
func (s *Stream) RemoteAddr() string {
	return s.session.conn.RemoteAddr().String()
}
