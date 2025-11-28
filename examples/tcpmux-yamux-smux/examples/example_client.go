// TCPMux 客户端示例 - 支持 IPv4/IPv6 选择
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"tcpmux-wrapper"
)

func main() {
	var (
		serverAddr = flag.String("addr", "localhost:8080", "TCPMux 服务器地址")
		muxType    = flag.String("mux", "yamux", "多路复用协议类型 (yamux/smux)")
		configFile = flag.String("config", "", "配置文件路径（JSON 格式）")
		useIPv6    = flag.Bool("ipv6", false, "使用 IPv6 连接目标站点")
		preferIPv6 = flag.Bool("prefer-ipv6", false, "优先使用 IPv6，失败则回退到 IPv4")
	)
	flag.Parse()

	var mux tcpmux.MuxType
	switch *muxType {
	case "yamux":
		mux = tcpmux.MuxTypeYamux
	case "smux":
		mux = tcpmux.MuxTypeSmux
	default:
		log.Fatalf("不支持的协议类型: %s，支持的类型: yamux, smux", *muxType)
	}

	// 加载配置
	config, err := LoadConfig(*configFile)
	if err != nil {
		log.Printf("加载配置失败，使用默认配置: %v", err)
		config = &ProxyConfig{
			UseIPv6:    *useIPv6,
			PreferIPv6: *preferIPv6,
		}
	}

	// 命令行参数覆盖配置文件
	if *useIPv6 {
		config.UseIPv6 = true
	}
	if *preferIPv6 {
		config.PreferIPv6 = true
	}

	log.Printf("配置: UseIPv6=%v, PreferIPv6=%v", config.UseIPv6, config.PreferIPv6)

	// 连接到服务器
	client, err := tcpmux.Dial(*serverAddr, mux)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	log.Printf("已连接到 TCPMux 服务器: %s (协议: %s)", *serverAddr, mux)
	log.Println("等待代理请求...")

	// 持续接受流（代理请求）- 服务端会主动打开流
	for {
		stream, err := client.Session().AcceptStream()
		if err != nil {
			if client.Session().IsClosed() {
				log.Println("会话已关闭")
				break
			}
			log.Printf("接受流失败: %v", err)
			continue
		}

		// 处理代理请求
		go handleProxyStream(stream, config)
	}
}

// handleProxyStream 处理代理流
func handleProxyStream(stream net.Conn, config *ProxyConfig) {
	defer stream.Close()

	log.Printf("收到代理请求流: %s -> %s", stream.RemoteAddr(), stream.LocalAddr())

	// 接收目标地址
	proxyReq, err := ReceiveProxyRequest(stream)
	if err != nil {
		log.Printf("接收代理请求失败: %v", err)
		return
	}

	targetAddr := fmt.Sprintf("%s:%s", proxyReq.TargetHost, proxyReq.TargetPort)
	log.Printf("目标地址: %s", targetAddr)

	// 根据配置选择 IPv4 或 IPv6 连接
	targetConn, err := dialTarget(targetAddr, config)
	if err != nil {
		log.Printf("连接目标失败 [%s]: %v", targetAddr, err)
		stream.Write([]byte(fmt.Sprintf("ERROR: %v\n", err)))
		return
	}
	defer targetConn.Close()

	log.Printf("已连接到目标: %s (使用 %s)", targetAddr, getNetworkType(targetConn))

	// 双向转发数据
	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(targetConn, stream)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(stream, targetConn)
		errCh <- err
	}()

	<-errCh
	log.Printf("代理流关闭: %s", targetAddr)
}

// dialTarget 根据配置连接目标
func dialTarget(targetAddr string, config *ProxyConfig) (net.Conn, error) {
	host, port, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return nil, err
	}

	// 如果强制使用 IPv6
	if config.UseIPv6 {
		return dialIPv6(host, port)
	}

	// 如果优先使用 IPv6
	if config.PreferIPv6 {
		conn, err := dialIPv6(host, port)
		if err == nil {
			return conn, nil
		}
		log.Printf("IPv6 连接失败，回退到 IPv4: %v", err)
	}

	// 默认使用 IPv4
	return dialIPv4(host, port)
}

// dialIPv4 使用 IPv4 连接
func dialIPv4(host, port string) (net.Conn, error) {
	// 解析 IPv4 地址
	ip := net.ParseIP(host)
	if ip != nil && ip.To4() != nil {
		// 直接是 IP 地址
		return net.Dial("tcp4", net.JoinHostPort(ip.String(), port))
	}

	// 域名解析，优先 IPv4
	addrs, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	// 查找 IPv4 地址
	for _, addr := range addrs {
		if addr.To4() != nil {
			return net.Dial("tcp4", net.JoinHostPort(addr.String(), port))
		}
	}

	return nil, fmt.Errorf("no IPv4 address found for %s", host)
}

// dialIPv6 使用 IPv6 连接
func dialIPv6(host, port string) (net.Conn, error) {
	// 解析 IPv6 地址
	ip := net.ParseIP(host)
	if ip != nil && ip.To16() != nil && ip.To4() == nil {
		// 直接是 IPv6 地址
		return net.Dial("tcp6", net.JoinHostPort("["+ip.String()+"]", port))
	}

	// 域名解析，查找 IPv6 地址
	addrs, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	// 查找 IPv6 地址
	for _, addr := range addrs {
		if addr.To4() == nil && addr.To16() != nil {
			return net.Dial("tcp6", net.JoinHostPort("["+addr.String()+"]", port))
		}
	}

	return nil, fmt.Errorf("no IPv6 address found for %s", host)
}

// getNetworkType 获取连接的网络类型
func getNetworkType(conn net.Conn) string {
	addr := conn.LocalAddr()
	if addr == nil {
		return "unknown"
	}

	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "unknown"
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return "unknown"
	}

	if ip.To4() != nil {
		return "IPv4"
	}

	if ip.To16() != nil {
		return "IPv6"
	}

	return "unknown"
}
