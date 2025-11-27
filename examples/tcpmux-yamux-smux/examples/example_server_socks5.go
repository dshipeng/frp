// TCPMux 服务端示例 - 使用 SOCKS5 协议（示例）
// 这个文件展示了如何在 TCPMux 流上使用 SOCKS5 协议替代自定义文本协议
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"tcpmux-wrapper"
)

// 注意：这是一个示例文件，展示如何使用 SOCKS5 协议
// 实际使用时，需要修改 example_server.go 中的 SendProxyRequest 调用

func handleProxyRequestWithSOCKS5(clientConn net.Conn, server *tcpmux.Server) {
	defer clientConn.Close()

	log.Printf("收到代理请求: %s", clientConn.RemoteAddr())

	// 解析代理请求，获取目标地址（这部分保持不变）
	proxyReq, err := ParseHTTPProxyRequest(clientConn)
	if err != nil {
		log.Printf("解析代理请求失败: %v", err)
		return
	}

	targetAddr := fmt.Sprintf("%s:%s", proxyReq.TargetHost, proxyReq.TargetPort)
	log.Printf("代理目标: %s", targetAddr)

	// 从服务端获取一个可用的客户端会话
	session := server.GetFirstAvailableSession()
	if session == nil {
		log.Printf("没有可用的客户端会话")
		clientConn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n"))
		return
	}

	log.Printf("使用客户端会话: %s -> %s", session.RemoteAddr(), session.LocalAddr())

	// 在客户端会话上打开新流（服务端主动发起）
	stream, err := server.OpenStreamOnSession(session)
	if err != nil {
		log.Printf("打开流失败: %v", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer stream.Close()

	log.Printf("已通过 TCPMux 打开流到客户端")

	// ============================================
	// 关键区别：使用 SOCKS5 协议替代自定义文本协议
	// ============================================
	// 旧方式（自定义文本）：
	//   SendProxyRequest(stream, proxyReq)  // 发送 "host:port\n"
	//
	// 新方式（SOCKS5）：
	if err := SendSOCKS5Request(stream, proxyReq.TargetHost, proxyReq.TargetPort); err != nil {
		log.Printf("发送 SOCKS5 请求失败: %v", err)
		return
	}

	// 等待 SOCKS5 响应
	// 注意：客户端需要发送 SOCKS5 响应
	socks5Resp, err := ReceiveSOCKS5Response(stream)
	if err != nil {
		log.Printf("接收 SOCKS5 响应失败: %v", err)
		return
	}

	if socks5Resp.Version != SOCKS5Version || socks5Resp.Rep != SOCKS5RepSuccess {
		log.Printf("SOCKS5 连接失败: version=%d, rep=%d", socks5Resp.Version, socks5Resp.Rep)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	log.Printf("SOCKS5 连接成功: %s:%d", socks5Resp.Address, socks5Resp.Port)

	// 如果是 CONNECT 方法，发送 200 响应
	if proxyReq.Method == "connect" {
		if err := HandleHTTPConnect(clientConn, targetAddr); err != nil {
			log.Printf("发送 CONNECT 响应失败: %v", err)
			return
		}
	}

	// 双向转发数据（这部分保持不变）
	log.Printf("开始转发数据: 用户 <-> TCPMux <-> 客户端 <-> 目标")
	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(stream, clientConn)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(clientConn, stream)
		errCh <- err
	}()

	<-errCh
	log.Printf("代理连接关闭: %s -> %s", clientConn.RemoteAddr(), targetAddr)
}
