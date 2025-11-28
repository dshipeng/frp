// TCPMux 服务端示例
package main

import (
	"io"
	"log"
	"net"
	"tcpmux"
)

func main() {
	// 创建服务器
	server, err := tcpmux.NewServer(":8080")
	if err != nil {
		log.Fatalf("创建服务器失败: %v", err)
	}

	// 设置流处理函数
	server.SetStreamHandler(func(stream *tcpmux.Stream) {
		log.Printf("处理新流: StreamID=%d, Local=%s, Remote=%s",
			stream.ID(), stream.LocalAddr(), stream.RemoteAddr())

		// 简单的代理服务器示例
		// 读取目标地址（简化格式：第一行是目标地址）
		buf := make([]byte, 1024)
		n, err := stream.Read(buf)
		if err != nil && err != io.EOF {
			log.Printf("读取失败: %v", err)
			stream.Close()
			return
		}

		if n == 0 {
			stream.Close()
			return
		}

		// 解析目标地址（简化：假设格式为 "host:port\n"）
		targetAddr := string(buf[:n])
		log.Printf("连接到目标: %s", targetAddr)

		// 连接到目标服务器
		targetConn, err := net.Dial("tcp", targetAddr)
		if err != nil {
			log.Printf("连接目标失败: %v", err)
			stream.Close()
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
		log.Printf("流 %d 关闭", stream.ID())
		stream.Close()
	})

	// 启动服务器
	log.Println("TCPMux 服务器启动，监听 :8080")
	if err := server.Serve(); err != nil {
		log.Fatalf("服务器错误: %v", err)
	}
}
