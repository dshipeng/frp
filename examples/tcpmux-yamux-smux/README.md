# 基于 yamux 和 smux 的 TCPMux 实现

本实现提供了对 yamux 和 smux 两种多路复用协议的统一封装，支持在运行时切换协议类型。

## 特性

- ✅ 支持 **yamux** 协议（HashiCorp 实现）
- ✅ 支持 **smux** 协议（高性能实现）
- ✅ 统一的接口，易于切换协议
- ✅ 完整的服务端和客户端实现
- ✅ 示例代码和性能测试

## 依赖

- [yamux](https://github.com/fatedier/yamux) - HashiCorp 的多路复用库
- [smux](https://github.com/xtaci/smux) - 高性能多路复用库

## 安装依赖

```bash
cd examples/tcpmux-yamux-smux
go mod tidy
```

## 快速开始

### 1. 启动服务端

```bash
cd examples
# 使用 yamux
go run example_server.go -addr :8080 -mux yamux

# 或使用 smux
go run example_server.go -addr :8080 -mux smux
```

### 2. 运行客户端

```bash
cd examples
# 使用 yamux
go run example_client.go -addr localhost:8080 -mux yamux -streams 5

# 或使用 smux
go run example_client.go -addr localhost:8080 -mux smux -streams 5
```

## 使用示例

### 服务端

```go
package main

import (
    "log"
    "net"
    "tcpmux-wrapper"
)

func main() {
    // 创建服务器（使用 yamux）
    server, err := tcpmux.NewServer(":8080", tcpmux.MuxTypeYamux)
    if err != nil {
        log.Fatal(err)
    }

    // 设置流处理函数
    server.SetStreamHandler(func(stream net.Conn) {
        defer stream.Close()
        
        // 处理流数据
        buf := make([]byte, 4096)
        n, _ := stream.Read(buf)
        stream.Write(buf[:n])
    })

    // 启动服务器
    server.Serve()
}
```

### 客户端

```go
package main

import (
    "log"
    "tcpmux-wrapper"
)

func main() {
    // 连接到服务器（使用 yamux）
    client, err := tcpmux.Dial("localhost:8080", tcpmux.MuxTypeYamux)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 打开流
    stream, err := client.OpenStream()
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()

    // 发送数据
    stream.Write([]byte("Hello"))
    
    // 读取响应
    buf := make([]byte, 1024)
    n, _ := stream.Read(buf)
    log.Printf("收到: %s", string(buf[:n]))
}
```

## 协议对比

### yamux

- **优点**：
  - 成熟稳定，被广泛使用（frp 使用）
  - 良好的流控制
  - 完善的错误处理

- **缺点**：
  - 性能相对较低
  - 内存占用较大

### smux

- **优点**：
  - 高性能，低延迟
  - 内存占用小
  - 适合高并发场景

- **缺点**：
  - 相对较新，生态不如 yamux

## 性能测试

运行性能测试：

```bash
# 先启动服务器
go run example_server.go -mux yamux &

# 运行性能测试
go run benchmark_test.go
```

## API 文档

### MuxType

协议类型枚举：

```go
const (
    MuxTypeYamux MuxType = "yamux"
    MuxTypeSmux MuxType = "smux"
)
```

### Server

```go
// 创建服务器
func NewServer(addr string, muxType MuxType) (*Server, error)

// 设置流处理函数
func (s *Server) SetStreamHandler(handler func(net.Conn))

// 启动服务
func (s *Server) Serve() error

// 关闭服务器
func (s *Server) Close() error
```

### Client

```go
// 连接到服务器
func Dial(addr string, muxType MuxType) (*Client, error)

// 打开新流
func (c *Client) OpenStream() (net.Conn, error)

// 关闭客户端
func (c *Client) Close() error
```

### MuxSession

```go
// 打开新流（客户端）
OpenStream() (net.Conn, error)

// 接受新流（服务端）
AcceptStream() (net.Conn, error)

// 关闭会话
Close() error

// 检查是否已关闭
IsClosed() bool

// 获取流数量
NumStreams() int
```

## 文件结构

```
tcpmux-yamux-smux/
├── go.mod              # Go 模块文件
├── mux_type.go        # 协议类型定义
├── mux_interface.go   # 接口定义
├── yamux_adapter.go   # yamux 适配器
├── smux_adapter.go    # smux 适配器
├── server.go          # 服务端实现
├── client.go          # 客户端实现
├── example_server.go  # 服务端示例
├── example_client.go  # 客户端示例
├── benchmark_test.go  # 性能测试
└── README.md          # 本文档
```

## 切换协议

切换协议非常简单，只需要在创建 Server 或 Client 时指定不同的 `MuxType`：

```go
// 使用 yamux
server, _ := tcpmux.NewServer(":8080", tcpmux.MuxTypeYamux)

// 切换到 smux
server, _ := tcpmux.NewServer(":8080", tcpmux.MuxTypeSmux)
```

## 注意事项

1. **协议一致性**：客户端和服务端必须使用相同的协议类型
2. **连接管理**：确保正确关闭流和会话，避免资源泄漏
3. **错误处理**：注意处理连接断开和流关闭的情况
4. **性能考虑**：根据实际场景选择合适的协议（yamux 更稳定，smux 性能更好）

## 扩展

如果需要添加新的多路复用协议支持：

1. 实现 `MuxFactory` 接口
2. 实现 `MuxSession` 接口
3. 在 `MuxType` 中添加新类型
4. 在 `NewServer` 和 `Dial` 中添加新协议的处理

## 许可证

本代码仅用于学习和演示目的。
