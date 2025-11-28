// TCPMux 服务端示例 - 带代理服务
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"tcpmux-wrapper"
)

// 注意：不再需要全局客户端连接池
// 服务端会维护所有连接到它的客户端会话

func main() {
	var (
		tcpmuxAddr = flag.String("tcpmux-addr", ":8080", "TCPMux 监听地址（客户端连接）")
		proxyAddr  = flag.String("proxy-addr", ":8888", "代理服务监听地址（用户连接）")
		muxType    = flag.String("mux", "yamux", "多路复用协议类型 (yamux/smux)")
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

	// 创建 TCPMux 服务器（用于客户端连接）
	tcpmuxServer, err := tcpmux.NewServer(*tcpmuxAddr, mux)
	if err != nil {
		log.Fatalf("创建 TCPMux 服务器失败: %v", err)
	}

	// 保存服务器引用，供代理服务使用
	var serverRef *tcpmux.Server = tcpmuxServer

	// 设置流处理函数 - 处理客户端发起的连接
	tcpmuxServer.SetStreamHandler(func(stream net.Conn) {
		defer stream.Close()

		log.Printf("客户端新流: %s -> %s", stream.RemoteAddr(), stream.LocalAddr())

		// 读取目标地址（格式：host:port\n）
		reader := bufio.NewReader(stream)
		targetAddr, _, err := reader.ReadLine()
		if err != nil {
			log.Printf("读取目标地址失败: %v", err)
			return
		}

		target := string(targetAddr)
		log.Printf("客户端请求连接目标: %s", target)

		// 连接到目标服务器
		targetConn, err := net.Dial("tcp", target)
		if err != nil {
			log.Printf("连接目标失败 [%s]: %v", target, err)
			stream.Write([]byte(fmt.Sprintf("ERROR: %v\n", err)))
			return
		}
		defer targetConn.Close()

		log.Printf("已连接到目标: %s", target)

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
		log.Printf("流关闭: %s", stream.RemoteAddr())
	})

	// 启动 TCPMux 服务器（在后台）
	go func() {
		log.Printf("TCPMux 服务器启动，监听 %s，使用协议: %s", *tcpmuxAddr, mux)
		if err := tcpmuxServer.Serve(); err != nil {
			log.Fatalf("TCPMux 服务器错误: %v", err)
		}
	}()

	// 创建代理服务器（用于用户连接）
	proxyListener, err := net.Listen("tcp", *proxyAddr)
	if err != nil {
		log.Fatalf("创建代理服务器失败: %v", err)
	}

	log.Printf("代理服务器启动，监听 %s", *proxyAddr)
	log.Println("等待客户端连接到 TCPMux 服务器...")

	// 接受代理请求
	for {
		clientConn, err := proxyListener.Accept()
		if err != nil {
			log.Printf("接受连接失败: %v", err)
			continue
		}

		go handleProxyRequest(clientConn, serverRef)
	}
}

// handleProxyRequest 处理代理请求
func handleProxyRequest(clientConn net.Conn, server *tcpmux.Server) {
	defer clientConn.Close()

	log.Printf("收到代理请求: %s", clientConn.RemoteAddr())

	// 解析代理请求，获取目标地址
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

	// 发送目标地址到客户端
	if err := SendProxyRequest(stream, proxyReq); err != nil {
		log.Printf("发送代理请求失败: %v", err)
		return
	}

	// 如果是 CONNECT 方法，发送 200 响应
	if proxyReq.Method == "connect" {
		if err := HandleHTTPConnect(clientConn, targetAddr); err != nil {
			log.Printf("发送 CONNECT 响应失败: %v", err)
			return
		}
	}

	// 双向转发数据
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
