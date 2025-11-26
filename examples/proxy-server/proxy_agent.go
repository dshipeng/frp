// 客户端代理服务器程序
// 接收来自 frp 的请求，并转发到目标服务器
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// 代理服务器监听地址
	proxyListenAddr = ":8080"
	// 连接超时时间
	connectTimeout = 30 * time.Second
)

func main() {
	log.Printf("启动代理服务器，监听地址: %s", proxyListenAddr)
	
	// 启动 HTTP 代理服务器
	server := &http.Server{
		Addr:         proxyListenAddr,
		Handler:      http.HandlerFunc(handleHTTPProxy),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("代理服务器已启动，等待连接...")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("代理服务器启动失败: %v", err)
	}
}

// handleHTTPProxy 处理 HTTP 代理请求
func handleHTTPProxy(w http.ResponseWriter, r *http.Request) {
	// 记录请求信息
	log.Printf("[%s] %s %s", r.Method, r.Host, r.URL.Path)

	// 处理 CONNECT 方法（用于 HTTPS 代理）
	if r.Method == http.MethodConnect {
		handleHTTPSProxy(w, r)
		return
	}

	// 处理普通 HTTP 请求
	handleHTTPRequest(w, r)
}

// handleHTTPSProxy 处理 HTTPS 代理（CONNECT 方法）
func handleHTTPSProxy(w http.ResponseWriter, r *http.Request) {
	// 获取目标地址
	targetAddr := r.Host
	if !strings.Contains(targetAddr, ":") {
		targetAddr += ":443"
	}

	log.Printf("CONNECT 请求: %s", targetAddr)

	// 连接到目标服务器
	targetConn, err := net.DialTimeout("tcp", targetAddr, connectTimeout)
	if err != nil {
		log.Printf("连接目标服务器失败 [%s]: %v", targetAddr, err)
		http.Error(w, fmt.Sprintf("连接失败: %v", err), http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	// 响应客户端，表示连接已建立
	w.WriteHeader(http.StatusOK)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// 获取底层连接
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Println("无法获取底层连接")
		http.Error(w, "不支持连接劫持", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		log.Printf("连接劫持失败: %v", err)
		return
	}
	defer clientConn.Close()

	// 双向转发数据
	log.Printf("开始转发数据: %s", targetAddr)
	errCh := make(chan error, 2)

	// 客户端 -> 目标服务器
	go func() {
		_, err := io.Copy(targetConn, clientConn)
		errCh <- err
	}()

	// 目标服务器 -> 客户端
	go func() {
		_, err := io.Copy(clientConn, targetConn)
		errCh <- err
	}()

	// 等待任一方向的数据传输完成
	<-errCh
	log.Printf("连接关闭: %s", targetAddr)
}

// handleHTTPRequest 处理普通 HTTP 请求
func handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// 解析目标 URL
	var targetURL *url.URL
	var err error

	// 如果是绝对 URL，直接使用
	if r.URL.IsAbs() {
		targetURL = r.URL
	} else {
		// 否则从 Host 头构建 URL
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		targetURL, err = url.Parse(fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path))
		if err != nil {
			log.Printf("解析 URL 失败: %v", err)
			http.Error(w, "无效的请求", http.StatusBadRequest)
			return
		}
		if r.URL.RawQuery != "" {
			targetURL.RawQuery = r.URL.RawQuery
		}
	}

	log.Printf("HTTP 请求: %s %s", r.Method, targetURL.String())

	// 创建新的请求
	req, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		log.Printf("创建请求失败: %v", err)
		http.Error(w, "创建请求失败", http.StatusInternalServerError)
		return
	}

	// 复制请求头
	for key, values := range r.Header {
		// 跳过一些不应该转发的头
		if strings.ToLower(key) == "connection" ||
			strings.ToLower(key) == "proxy-connection" ||
			strings.ToLower(key) == "keep-alive" {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: connectTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, connectTimeout)
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 不自动跟随重定向，让客户端处理
			return http.ErrUseLastResponse
		},
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("请求失败 [%s]: %v", targetURL.String(), err)
		http.Error(w, fmt.Sprintf("请求失败: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 设置状态码
	w.WriteHeader(resp.StatusCode)

	// 复制响应体
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("复制响应体失败: %v", err)
	}

	log.Printf("请求完成: %s [%d]", targetURL.String(), resp.StatusCode)
}

// handleSOCKS5Proxy 处理 SOCKS5 代理（可选实现）
// 注意：当前主函数只启动 HTTP 代理，如需 SOCKS5 支持，需要单独启动 SOCKS5 服务器
func handleSOCKS5Proxy(conn net.Conn) {
	defer conn.Close()

	// SOCKS5 握手
	buf := make([]byte, 256)
	
	// 读取客户端认证方法
	n, err := conn.Read(buf)
	if err != nil || n < 3 {
		log.Printf("读取 SOCKS5 握手失败: %v", err)
		return
	}

	// 检查版本号
	if buf[0] != 0x05 {
		log.Printf("不支持的 SOCKS 版本: %d", buf[0])
		return
	}

	// 支持的认证方法数量
	methodCount := int(buf[1])
	if n < 2+methodCount {
		log.Println("SOCKS5 握手数据不完整")
		return
	}

	// 选择无需认证的方法
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		log.Printf("写入 SOCKS5 响应失败: %v", err)
		return
	}

	// 读取连接请求
	n, err = conn.Read(buf)
	if err != nil || n < 10 {
		log.Printf("读取 SOCKS5 连接请求失败: %v", err)
		return
	}

	// 解析目标地址
	var targetAddr string
	switch buf[3] {
	case 0x01: // IPv4
		if n < 10 {
			log.Println("SOCKS5 IPv4 地址不完整")
			return
		}
		targetAddr = fmt.Sprintf("%d.%d.%d.%d:%d",
			buf[4], buf[5], buf[6], buf[7],
			uint16(buf[8])<<8|uint16(buf[9]))
	case 0x03: // 域名
		domainLen := int(buf[4])
		if n < 5+domainLen+2 {
			log.Println("SOCKS5 域名地址不完整")
			return
		}
		domain := string(buf[5 : 5+domainLen])
		port := uint16(buf[5+domainLen])<<8 | uint16(buf[5+domainLen+1])
		targetAddr = fmt.Sprintf("%s:%d", domain, port)
	default:
		log.Printf("不支持的地址类型: %d", buf[3])
		conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}

	log.Printf("SOCKS5 连接请求: %s", targetAddr)

	// 连接到目标服务器
	targetConn, err := net.DialTimeout("tcp", targetAddr, connectTimeout)
	if err != nil {
		log.Printf("连接目标服务器失败 [%s]: %v", targetAddr, err)
		conn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}
	defer targetConn.Close()

	// 响应连接成功
	if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
		log.Printf("写入 SOCKS5 连接响应失败: %v", err)
		return
	}

	// 双向转发数据
	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(targetConn, conn)
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(conn, targetConn)
		errCh <- err
	}()

	<-errCh
	log.Printf("SOCKS5 连接关闭: %s", targetAddr)
}
