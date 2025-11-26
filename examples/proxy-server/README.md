# 基于 frp TCPMux 的代理服务示例

本示例演示如何使用 frp 的 TCPMux 功能实现一个简易的内网穿透代理服务。

## 架构说明

```
客户端（内网）                   服务端（公网）                   目标服务器
┌─────────────┐                ┌─────────────┐                ┌─────────────┐
│             │                │             │                │             │
│ 代理服务器   │  ←─TCPMux──→   │   frps      │  ←─TCP──→      │  目标网站    │
│ (proxy_agent)│                │             │                │             │
│             │                │             │                │             │
└─────────────┘                └─────────────┘                └─────────────┘
    :8080                           :8888
```

## 工作流程

1. **客户端代理服务器** (`proxy_agent.go`) 在客户端内网启动，监听 `8080` 端口
2. **frpc** 通过 TCPMux 连接到 **frps**，将本地 `8080` 端口映射到服务端的 `8888` 端口
3. 用户通过服务端的 `8888` 端口发送代理请求
4. 请求通过 frp 隧道转发到客户端的代理服务器
5. 客户端代理服务器将请求转发到目标服务器（使用客户端的默认网络）

## 文件说明

- `frps.toml`: frp 服务端配置文件（启用 TCPMux）
- `frpc.toml`: frp 客户端配置文件（启用 TCPMux，配置 TCP 代理）
- `proxy_agent.go`: 客户端代理服务器程序（HTTP/HTTPS 代理）

## 快速开始

### 前置要求

- Go 1.21+ （用于编译代理服务器）
- frps 和 frpc 二进制文件（从 [frp releases](https://github.com/fatedier/frp/releases) 下载）

### 使用步骤

#### 1. 配置服务端

编辑 `frps.toml`，修改以下配置：

```toml
bindAddr = "0.0.0.0"  # 服务端监听地址
bindPort = 7000        # 服务端监听端口
auth.token = "your_secret_token_here"  # 设置你的密钥
```

在服务端启动 frps：

```bash
frps -c frps.toml
```

#### 2. 配置客户端

编辑 `frpc.toml`，修改以下配置：

```toml
serverAddr = "your_frps_server_ip"  # 修改为你的 frps 服务器地址（IP 或域名）
serverPort = 7000
auth.token = "your_secret_token_here"  # 必须与服务端一致
```

#### 3. 编译并启动代理服务器

在客户端机器上编译代理服务器：

```bash
cd examples/proxy-server
go build -o proxy_agent proxy_agent.go
```

启动代理服务器：

```bash
./proxy_agent
```

你应该看到类似输出：
```
启动代理服务器，监听地址: :8080
代理服务器已启动，等待连接...
```

#### 4. 启动 frpc

在客户端机器上启动 frpc：

```bash
frpc -c frpc.toml
```

如果连接成功，你应该看到类似输出：
```
login to server success, get run id [xxx]
```

#### 5. 使用代理

现在你可以通过服务端的 `8888` 端口使用代理服务了。

#### HTTP 代理

```bash
# 使用 curl（替换 your_frps_server 为实际的服务端地址）
curl -x http://your_frps_server:8888 http://example.com

# 使用环境变量
export http_proxy=http://your_frps_server:8888
export https_proxy=http://your_frps_server:8888
curl http://example.com
```

#### HTTPS 代理（CONNECT 方法）

代理服务器自动支持 HTTPS 代理，使用 CONNECT 方法：

```bash
curl -x http://your_frps_server:8888 https://example.com
```

#### 浏览器配置

在浏览器中配置代理服务器：
- **代理类型**：HTTP
- **代理地址**：`your_frps_server`（或 IP 地址）
- **代理端口**：`8888`

#### 使用测试脚本

项目提供了测试脚本，可以快速验证代理是否工作：

```bash
# 设置服务端地址（如果不在本地）
export FRPS_HOST=your_frps_server

# 运行测试
./test_proxy.sh
```

## 特性

- ✅ 支持 HTTP 代理
- ✅ 支持 HTTPS 代理（CONNECT 方法）
- ✅ 使用 TCPMux 减少连接数
- ✅ 自动转发请求头
- ✅ 连接超时控制
- ✅ 详细的日志记录

## 注意事项

1. **安全性**：本示例仅用于演示，生产环境请：
   - 使用 TLS 加密传输
   - 添加身份认证
   - 限制访问 IP
   - 使用强密码

2. **性能**：TCPMux 可以减少连接数，但所有流量都通过单条 TCP 连接，可能成为瓶颈

3. **防火墙**：确保服务端防火墙开放相应端口（7000, 8888）

4. **网络**：客户端需要能够访问目标服务器（使用客户端的默认网络）

## 扩展功能

### 添加 SOCKS5 支持

代码中已包含 `handleSOCKS5Proxy` 函数，可以扩展支持 SOCKS5 代理。

### 添加认证

可以在代理服务器中添加 HTTP Basic Auth 或 Token 认证。

### 添加日志和监控

可以集成日志系统和监控系统，记录代理使用情况。

## 故障排查

1. **连接失败**：检查 frps 和 frpc 是否正常启动
2. **认证失败**：检查 token 是否一致
3. **端口占用**：检查端口是否被占用
4. **网络不通**：检查客户端是否能访问目标服务器

查看日志：

```bash
# frps 日志
tail -f frps.log

# frpc 日志
tail -f frpc.log

# 代理服务器日志
# 直接查看控制台输出
```
