// Session 管理多路复用的 TCP 连接
package tcpmux

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
)

// Session 表示一个 TCPMux 会话
type Session struct {
	conn        net.Conn
	streams     map[uint32]*Stream
	streamsMu   sync.RWMutex
	nextID      uint32
	isServer    bool
	closed      int32
	closeCh     chan struct{}
	newStreamCh chan *Stream // 新流通知通道（服务端使用）
}

// NewSession 创建新的会话
func NewSession(conn net.Conn, isServer bool) *Session {
	s := &Session{
		conn:        conn,
		streams:     make(map[uint32]*Stream),
		isServer:    isServer,
		closeCh:     make(chan struct{}),
		newStreamCh: make(chan *Stream, 10),
		nextID:      1,
	}

	// 启动读取协程
	go s.readLoop()

	return s
}

// OpenStream 打开一个新的流（客户端）
func (s *Session) OpenStream() (*Stream, error) {
	if atomic.LoadInt32(&s.closed) == 1 {
		return nil, io.ErrClosedPipe
	}

	// 分配新的流 ID
	streamID := atomic.AddUint32(&s.nextID, 1)
	if streamID == 0 {
		streamID = atomic.AddUint32(&s.nextID, 1) // 跳过 0
	}

	stream := &Stream{
		id:      streamID,
		session: s,
		readCh:  make(chan []byte, 10),
		writeCh: make(chan []byte, 10),
		closeCh: make(chan struct{}),
	}

	// 注册流
	s.streamsMu.Lock()
	s.streams[streamID] = stream
	s.streamsMu.Unlock()

	// 发送 SYN 帧
	frame := &Frame{
		Type:     FrameTypeData,
		Flags:    FlagSYN,
		StreamID: streamID,
		Length:   0,
	}

	if err := WriteFrame(s.conn, frame); err != nil {
		s.streamsMu.Lock()
		delete(s.streams, streamID)
		s.streamsMu.Unlock()
		return nil, err
	}

	return stream, nil
}

// AcceptStream 接受一个新的流（服务端）
func (s *Session) AcceptStream() (*Stream, error) {
	if !s.isServer {
		return nil, errors.New("AcceptStream only available on server side")
	}

	select {
	case stream := <-s.newStreamCh:
		return stream, nil
	case <-s.closeCh:
		return nil, errors.New("session closed")
	}
}

// readLoop 读取循环，处理所有接收到的帧
func (s *Session) readLoop() {
	defer s.Close()

	for {
		frame, err := DecodeFrame(s.conn)
		if err != nil {
			if err != io.EOF {
				// 记录错误（简化实现）
			}
			return
		}

		s.handleFrame(frame)
	}
}

// handleFrame 处理接收到的帧
func (s *Session) handleFrame(frame *Frame) {
	s.streamsMu.RLock()
	stream, streamExists := s.streams[frame.StreamID]
	s.streamsMu.RUnlock()

	switch frame.Type {
	case FrameTypeData:
		if frame.Flags&FlagSYN != 0 {
			// 新建流（服务端）
			if !s.isServer {
				return // 客户端不应该收到 SYN
			}

			// 检查流是否已存在
			if !streamExists {
				stream = &Stream{
					id:      frame.StreamID,
					session: s,
					readCh:  make(chan []byte, 10),
					writeCh: make(chan []byte, 10),
					closeCh: make(chan struct{}),
				}

				s.streamsMu.Lock()
				s.streams[frame.StreamID] = stream
				s.streamsMu.Unlock()

				// 通知新流创建
				select {
				case s.newStreamCh <- stream:
				default:
					// 通道满，记录警告（简化处理）
				}
			}
		}

		if stream != nil {
			// 发送数据到流
			select {
			case stream.readCh <- frame.Data:
			default:
				// 缓冲区满（简化处理）
			}
		}

		if frame.Flags&FlagFIN != 0 {
			// 流结束
			if stream != nil {
				stream.closeMu.Lock()
				if !stream.closed {
					stream.closed = true
					close(stream.closeCh)
				}
				stream.closeMu.Unlock()

				s.streamsMu.Lock()
				delete(s.streams, frame.StreamID)
				s.streamsMu.Unlock()
			}
		}

	case FrameTypePing:
		// 响应 Pong
		pongFrame := &Frame{
			Type:     FrameTypePong,
			StreamID: 0,
			Length:   0,
		}
		WriteFrame(s.conn, pongFrame)

	case FrameTypeGoAway:
		// 关闭会话
		s.Close()
	}
}

// writeStream 写入流数据
func (s *Session) writeStream(streamID uint32, data []byte) (int, error) {
	if atomic.LoadInt32(&s.closed) == 1 {
		return 0, io.ErrClosedPipe
	}

	// 分片发送（如果数据太大）
	offset := 0
	total := len(data)

	for offset < total {
		chunkSize := total - offset
		if chunkSize > MaxFrameSize {
			chunkSize = MaxFrameSize
		}

		frame := &Frame{
			Type:     FrameTypeData,
			Flags:    0,
			StreamID: streamID,
			Length:   uint32(chunkSize),
			Data:     data[offset : offset+chunkSize],
		}

		if err := WriteFrame(s.conn, frame); err != nil {
			return offset, err
		}

		offset += chunkSize
	}

	return total, nil
}

// closeStream 关闭流
func (s *Session) closeStream(streamID uint32) error {
	frame := &Frame{
		Type:     FrameTypeData,
		Flags:    FlagFIN,
		StreamID: streamID,
		Length:   0,
	}

	if err := WriteFrame(s.conn, frame); err != nil {
		return err
	}

	s.streamsMu.Lock()
	delete(s.streams, streamID)
	s.streamsMu.Unlock()

	return nil
}

// Close 关闭会话
func (s *Session) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil
	}

	// 发送 GoAway 帧
	frame := &Frame{
		Type:     FrameTypeGoAway,
		StreamID: 0,
		Length:   0,
	}
	WriteFrame(s.conn, frame)

	// 关闭所有流
	s.streamsMu.Lock()
	for _, stream := range s.streams {
		stream.closeMu.Lock()
		if !stream.closed {
			stream.closed = true
			close(stream.closeCh)
		}
		stream.closeMu.Unlock()
	}
	s.streams = make(map[uint32]*Stream)
	s.streamsMu.Unlock()

	close(s.closeCh)
	return s.conn.Close()
}

// IsClosed 检查会话是否已关闭
func (s *Session) IsClosed() bool {
	return atomic.LoadInt32(&s.closed) == 1
}
