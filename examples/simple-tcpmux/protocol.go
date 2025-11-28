// TCPMux 协议定义
// 简易实现，用于演示 TCP 流多路复用的基本原理

package tcpmux

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	// 帧类型
	FrameTypeData   = 0x01 // 数据帧
	FrameTypeWindow = 0x02 // 窗口更新帧
	FrameTypePing   = 0x03 // Ping 帧
	FrameTypePong   = 0x04 // Pong 帧
	FrameTypeGoAway = 0x05 // 关闭连接帧

	// 标志位
	FlagSYN = 0x01 // 同步标志（新建流）
	FlagFIN = 0x02 // 结束标志（关闭流）
	FlagACK = 0x04 // 确认标志

	// 最大帧大小（不包括头部）
	MaxFrameSize = 64 * 1024 // 64KB

	// 头部大小
	HeaderSize = 12
)

var (
	// 协议魔数，用于检测协议版本
	Magic = [4]byte{0x54, 0x43, 0x50, 0x4D} // "TCPM"
)

// Frame 表示一个 TCPMux 帧
type Frame struct {
	Type      uint8  // 帧类型
	Flags     uint8  // 标志位
	StreamID  uint32 // 流 ID
	Length    uint32 // 数据长度
	Data      []byte // 数据（可选）
}

// Encode 将帧编码为字节流
func (f *Frame) Encode() []byte {
	buf := make([]byte, HeaderSize+len(f.Data))
	
	// 魔数（4字节）
	copy(buf[0:4], Magic[:])
	
	// 类型和标志（1字节 + 1字节）
	buf[4] = f.Type
	buf[5] = f.Flags
	
	// 流 ID（4字节，大端序）
	binary.BigEndian.PutUint32(buf[6:10], f.StreamID)
	
	// 数据长度（4字节，大端序）
	binary.BigEndian.PutUint32(buf[10:14], uint32(len(f.Data)))
	
	// 数据（如果有）
	if len(f.Data) > 0 {
		copy(buf[14:], f.Data)
	}
	
	return buf
}

// Decode 从字节流解码帧
func DecodeFrame(r io.Reader) (*Frame, error) {
	// 读取头部
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	
	// 检查魔数
	if header[0] != Magic[0] || header[1] != Magic[1] ||
		header[2] != Magic[2] || header[3] != Magic[3] {
		return nil, errors.New("invalid magic number")
	}
	
	frame := &Frame{
		Type:     header[4],
		Flags:    header[5],
		StreamID: binary.BigEndian.Uint32(header[6:10]),
		Length:   binary.BigEndian.Uint32(header[10:14]),
	}
	
	// 读取数据（如果有）
	if frame.Length > 0 {
		if frame.Length > MaxFrameSize {
			return nil, errors.New("frame too large")
		}
		frame.Data = make([]byte, frame.Length)
		if _, err := io.ReadFull(r, frame.Data); err != nil {
			return nil, err
		}
	}
	
	return frame, nil
}

// WriteFrame 将帧写入 Writer
func WriteFrame(w io.Writer, frame *Frame) error {
	data := frame.Encode()
	_, err := w.Write(data)
	return err
}
