# 简易 TCPMux 实现

这是一个简易的 TCPMux（TCP 流多路复用）实现，用于演示在单条 TCP 连接上创建多个逻辑流的基本原理。

## 什么是 TCPMux？

TCPMux（TCP Multiplexing）是一种在单条 TCP 连接上多路复用多个逻辑流的技术。它的主要优势包括：

- **减少连接数**：多个逻辑流共享一条 TCP 连接
- **降低开销**：避免频繁的 TCP 握手和挥手
- **提高效率**：特别适合需要大量并发连接的场景

## 架构设计

### 协议格式

```
+--------+--------+--------+--------+--------+--------+--------+
| Magic  | Type   | Flags  | StreamID (4 bytes) | Length (4 bytes) |
+--------+--------+--------+--------+--------+--------+--------+
|                    Data (Length bytes)                        |
+---------------------------------------------------------------+
```

- **Magic** (4 bytes): 协议魔数 "TCPM"
- **Type** (1 byte): 帧类型（数据、窗口更新、Ping/Pong、GoAway）
- **Flags** (1 byte): 标志位（SYN、FIN、ACK）
- **StreamID** (4 bytes): 流标识符
- **Length** (4 bytes): 数据长度
- **Data**: 实际数据（可选）

### 核心组件

1. **Frame**: 协议帧，封装数据和控制信息
2. **Stream**: 逻辑流，提供类似 net.Conn 的接口
3. **Session**: 会话管理，处理多路复用逻辑
4. **Server/Client**: 服务端和客户端实现

## 文件结构

```
simple-tcpmux/
├── protocol.go      # 协议定义和编解码
├── stream.go        # 流实现
├── session.go       # 会话管理
├── server.go       # 服务端实现
├── client.go        # 客户端实现
├── example_server.go # 服务端示例
├── example_client.go # 客户端示例
├── go.mod          # Go 模块文件
└── README.md       # 本文档
```

## 快速开始

### 1. 编译

```bash
cd examples/simple-tcpmux
go mod tidy
go build -o server example_server.go
go build -o client example_client.go
```

### 2. 启动服务端

```bash
./server
```

服务端将监听 `:8080` 端口。

### 3. 启动客户端

```bash
./client localhost:8080
```

客户端将连接到服务器，创建多个流并发送测试数据。

## 使用示例

### 基本用法

#### 服务端

```go
package main

import (
    "log"
    "tcpmux"
)

func main() {
    server, err := tcpmux.NewServer(":8080")
    if err != nil {
        log.Fatal(err)
    }

    server.SetStreamHandler(func(stream *tcpmux.Stream) {
        // 处理流
        buf := make([]byte, 1024)
        n, _ := stream.Read(buf)
        stream.Write(buf[:n])
        stream.Close()
    })

    server.Serve()
}
```

#### 客户端

```go
package main

import (
    "log"
    "tcpmux"
)

func main() {
    client, err := tcpmux.Dial("localhost:8080")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    stream, err := client.OpenStream()
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()

    stream.Write([]byte("Hello"))
    buf := make([]byte, 1024)
    n, _ := stream.Read(buf)
    log.Printf("收到: %s", string(buf[:n]))
}
```

## 协议细节

### 帧类型

- `FrameTypeData` (0x01): 数据帧
- `FrameTypeWindow` (0x02): 窗口更新帧（未实现）
- `FrameTypePing` (0x03): Ping 帧
- `FrameTypePong` (0x04): Pong 帧
- `FrameTypeGoAway` (0x05): 关闭连接帧

### 标志位

- `FlagSYN` (0x01): 同步标志，用于创建新流
- `FlagFIN` (0x02): 结束标志，用于关闭流
- `FlagACK` (0x04): 确认标志（未实现）

### 流生命周期

1. **创建流**：客户端发送 SYN 帧
2. **数据传输**：通过数据帧传输
3. **关闭流**：发送 FIN 帧

## 实现特点

### 已实现功能

- ✅ 基本的帧协议
- ✅ 流创建和管理
- ✅ 数据读写
- ✅ 流关闭
- ✅ 会话管理
- ✅ 服务端和客户端

### 未实现功能（简化版本）

- ❌ 流控制（窗口管理）
- ❌ 拥塞控制
- ❌ 优先级
- ❌ 错误恢复
- ❌ 加密
- ❌ 压缩

## 与 yamux 的对比

| 特性 | 本实现 | yamux |
|------|--------|-------|
| 协议复杂度 | 简单 | 复杂 |
| 流控制 | 无 | 有 |
| 拥塞控制 | 无 | 有 |
| 性能 | 基础 | 优化 |
| 适用场景 | 学习/演示 | 生产环境 |

## 性能考虑

本实现是简化版本，主要用于学习和演示。生产环境建议使用：

- [yamux](https://github.com/hashicorp/yamux) - HashiCorp 的多路复用库
- [smux](https://github.com/xtaci/smux) - 高性能多路复用库
- [quic-go](https://github.com/quic-go/quic-go) - 基于 UDP 的多路复用

## 扩展建议

如果需要改进本实现，可以考虑：

1. **添加流控制**：实现滑动窗口机制
2. **添加优先级**：支持流的优先级调度
3. **添加加密**：集成 TLS 或自定义加密
4. **添加压缩**：支持数据压缩
5. **改进错误处理**：更完善的错误恢复机制
6. **性能优化**：减少内存分配，优化数据拷贝

## 故障排查

### 连接失败

- 检查服务端是否启动
- 检查端口是否被占用
- 检查防火墙设置

### 数据丢失

- 本实现是简化版本，没有流控制
- 大数据传输可能需要分片处理

### 流创建失败

- 检查会话是否已关闭
- 检查流 ID 是否冲突

## 许可证

本代码仅用于学习和演示目的。
