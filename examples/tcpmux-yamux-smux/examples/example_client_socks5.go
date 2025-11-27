// TCPMux 客户端示例 - 使用 SOCKS5 协议（示例）
// 这个文件展示了客户端如何接收和处理 SOCKS5 协议
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
// 实际使用时，需要修改 example_client.go 中的 ReceiveProxyRequest 调用

func handleProxyStreamWithSOCKS5(stream net.Conn, config *ProxyConfig) {
	defer stream.Close()

	log.Printf("收到代理请求流: %s -> %s", stream.RemoteAddr(), stream.LocalAddr())

	// ============================================
	// 关键区别：使用 SOCKS5 协议替代自定义文本协议
	// ============================================
	// 旧方式（自定义文本）：
	//   proxyReq, err := ReceiveProxyRequest(stream)  // 读取 "host:port\n"
	//
	// 新方式（SOCKS5）：
	socks5Req, err := ReceiveSOCKS5Request(stream)
	if err != nil {
		log.Printf("接收 SOCKS5 请求失败: %v", err)
		// 发送错误响应
		SendSOCKS5Response(stream, SOCKS5RepFailure, "", 0)
		return
	}

	log.Printf("收到 SOCKS5 请求: %s:%d (CMD=%d)", socks5Req.Address, socks5Req.Port, socks5Req.Command)

	// 根据配置选择 IPv4 或 IPv6 连接目标
	targetAddr := fmt.Sprintf("%s:%d", socks5Req.Address, socks5Req.Port)
	targetConn, err := dialTarget(targetAddr, config)
	if err != nil {
		log.Printf("连接目标失败 [%s]: %v", targetAddr, err)
		// 发送 SOCKS5 错误响应
		SendSOCKS5Response(stream, SOCKS5RepFailure, "", 0)
		return
	}
	defer targetConn.Close()

	log.Printf("已连接到目标: %s (使用 %s)", targetAddr, getNetworkType(targetConn))

	// 发送 SOCKS5 成功响应
	// 注意：这里简化处理，实际应该返回绑定的地址和端口
	if err := SendSOCKS5Response(stream, SOCKS5RepSuccess, "0.0.0.0", 0); err != nil {
		log.Printf("发送 SOCKS5 响应失败: %v", err)
		return
	}

	// 双向转发数据（这部分保持不变）
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
