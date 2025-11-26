// 代理协议处理
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// ProxyRequest 代理请求信息
type ProxyRequest struct {
	TargetHost string
	TargetPort string
	Method     string // "http" 或 "connect"
}

// ParseHTTPProxyRequest 解析 HTTP 代理请求
func ParseHTTPProxyRequest(conn net.Conn) (*ProxyRequest, error) {
	reader := bufio.NewReader(conn)
	
	// 读取第一行
	line, _, err := reader.ReadLine()
	if err != nil {
		return nil, err
	}

	// 解析请求行
	parts := strings.Split(string(line), " ")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid request line: %s", string(line))
	}

	method := parts[0]
	requestURL := parts[1]

	// 如果是 CONNECT 方法
	if method == "CONNECT" {
		// CONNECT host:port HTTP/1.1
		hostPort := requestURL
		return parseHostPort(hostPort, "connect")
	}

	// 如果是普通 HTTP 请求
	if method == "GET" || method == "POST" || method == "PUT" || method == "DELETE" {
		// 解析 URL
		if strings.HasPrefix(requestURL, "http://") || strings.HasPrefix(requestURL, "https://") {
			// 绝对 URL
			u, err := url.Parse(requestURL)
			if err != nil {
				return nil, err
			}
			host, port := splitHostPort(u.Host)
			return &ProxyRequest{
				TargetHost: host,
				TargetPort: port,
				Method:     "http",
			}, nil
		}

		// 相对 URL，需要从 Host 头获取
		// 读取所有头部
		req, err := http.ReadRequest(reader)
		if err != nil {
			return nil, err
		}

		host, port := splitHostPort(req.Host)
		return &ProxyRequest{
			TargetHost: host,
			TargetPort: port,
			Method:     "http",
		}, nil
	}

	return nil, fmt.Errorf("unsupported method: %s", method)
}

// parseHostPort 解析 host:port
func parseHostPort(hostPort, method string) (*ProxyRequest, error) {
	host, port := splitHostPort(hostPort)
	return &ProxyRequest{
		TargetHost: host,
		TargetPort: port,
		Method:     method,
	}, nil
}

// splitHostPort 分离主机和端口
func splitHostPort(hostPort string) (host, port string) {
	if strings.Contains(hostPort, ":") {
		parts := strings.Split(hostPort, ":")
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		// IPv6 地址
		if strings.Contains(hostPort, "[") {
			// [::1]:8080 格式
			idx := strings.LastIndex(hostPort, ":")
			if idx > 0 {
				host = hostPort[:idx]
				port = hostPort[idx+1:]
				// 移除 IPv6 地址的方括号
				host = strings.Trim(host, "[]")
				return
			}
		}
	}
	
	// 默认端口
	if port == "" {
		port = "80"
	}
	return hostPort, port
}

// SendProxyRequest 通过 tcpmux 发送代理请求
func SendProxyRequest(stream net.Conn, req *ProxyRequest) error {
	// 发送目标地址（格式：host:port\n）
	targetAddr := fmt.Sprintf("%s:%s\n", req.TargetHost, req.TargetPort)
	_, err := stream.Write([]byte(targetAddr))
	return err
}

// ReceiveProxyRequest 从 tcpmux 接收代理请求
func ReceiveProxyRequest(stream net.Conn) (*ProxyRequest, error) {
	reader := bufio.NewReader(stream)
	
	// 读取目标地址（格式：host:port\n）
	line, _, err := reader.ReadLine()
	if err != nil {
		return nil, err
	}

	targetAddr := string(line)
	host, port := splitHostPort(targetAddr)
	
	return &ProxyRequest{
		TargetHost: host,
		TargetPort: port,
		Method:     "connect", // 默认使用 CONNECT 方法
	}, nil
}

// HandleHTTPConnect 处理 HTTP CONNECT 请求
func HandleHTTPConnect(clientConn net.Conn, targetAddr string) error {
	// 发送 200 Connection Established
	response := "HTTP/1.1 200 Connection Established\r\n\r\n"
	if _, err := clientConn.Write([]byte(response)); err != nil {
		return err
	}

	return nil
}

// HandleHTTPProxy 处理普通 HTTP 代理请求
func HandleHTTPProxy(clientConn net.Conn, req *http.Request) error {
	// 这里可以记录请求，但不需要特殊处理
	// 数据会直接转发
	return nil
}

// CopyData 双向复制数据
func CopyData(dst, src net.Conn) error {
	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(dst, src)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(src, dst)
		errCh <- err
	}()

	// 等待任一方向完成
	err := <-errCh
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

// ReadHTTPRequest 从连接读取 HTTP 请求（保留原始数据）
func ReadHTTPRequest(conn net.Conn) ([]byte, *http.Request, error) {
	reader := bufio.NewReader(conn)
	
	// 读取请求行
	requestLine, _, err := reader.ReadLine()
	if err != nil {
		return nil, nil, err
	}

	var buf bytes.Buffer
	buf.Write(requestLine)
	buf.WriteString("\r\n")

	// 读取头部
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			return nil, nil, err
		}

		buf.Write(line)
		buf.WriteString("\r\n")

		// 空行表示头部结束
		if len(line) == 0 {
			break
		}
	}

	// 解析请求
	req, err := http.ReadRequest(bufio.NewReader(&buf))
	if err != nil {
		return nil, nil, err
	}

	return buf.Bytes(), req, nil
}
