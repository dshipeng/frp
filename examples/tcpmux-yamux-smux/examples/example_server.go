// TCPMux 服务端示例
package main

import (
	"flag"
	"io"
	"log"
	"net"
	"tcpmux-wrapper"
)

func main() {
	var (
		addr    = flag.String("addr", ":8080", "监听地址")
		muxType = flag.String("mux", "yamux", "多路复用协议类型 (yamux/smux)")
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

	// 创建服务器
	server, err := tcpmux.NewServer(*addr, mux)
	if err != nil {
		log.Fatalf("创建服务器失败: %v", err)
	}

	// 设置流处理函数 - 简单的代理服务器示例
	server.SetStreamHandler(func(stream net.Conn) {
		defer stream.Close()

		log.Printf("处理新流: %s -> %s", stream.RemoteAddr(), stream.LocalAddr())

		// 读取目标地址（简化格式：第一行是目标地址 "host:port\n"）
		buf := make([]byte, 1024)
		n, err := stream.Read(buf)
		if err != nil && err != io.EOF {
			log.Printf("读取失败: %v", err)
			return
		}

		if n == 0 {
			return
		}

		// 解析目标地址
		targetAddr := string(buf[:n])
		// 移除换行符
		if len(targetAddr) > 0 && targetAddr[len(targetAddr)-1] == '\n' {
			targetAddr = targetAddr[:len(targetAddr)-1]
		}
		if len(targetAddr) > 0 && targetAddr[len(targetAddr)-1] == '\r' {
			targetAddr = targetAddr[:len(targetAddr)-1]
		}

		log.Printf("连接到目标: %s", targetAddr)

		// 连接到目标服务器
		targetConn, err := net.Dial("tcp", targetAddr)
		if err != nil {
			log.Printf("连接目标失败: %v", err)
			return
		}
		defer targetConn.Close()

		// 双向转发
		errCh := make(chan error, 2)

		// 流 -> 目标
		go func() {
			_, err := io.Copy(targetConn, stream)
			errCh <- err
		}()

		// 目标 -> 流
		go func() {
			_, err := io.Copy(stream, targetConn)
			errCh <- err
		}()

		// 等待任一方向完成
		<-errCh
		log.Printf("流关闭: %s", stream.RemoteAddr())
	})

	// 启动服务器
	log.Printf("TCPMux 服务器启动，监听 %s，使用协议: %s", *addr, mux)
	if err := server.Serve(); err != nil {
		log.Fatalf("服务器错误: %v", err)
	}
}
