// 多路复用协议类型定义
package tcpmux

// MuxType 表示多路复用协议类型
type MuxType string

const (
	// MuxTypeYamux 使用 yamux 协议
	MuxTypeYamux MuxType = "yamux"
	// MuxTypeSmux 使用 smux 协议
	MuxTypeSmux MuxType = "smux"
)

// String 返回协议类型的字符串表示
func (t MuxType) String() string {
	return string(t)
}

// IsValid 检查协议类型是否有效
func (t MuxType) IsValid() bool {
	return t == MuxTypeYamux || t == MuxTypeSmux
}
