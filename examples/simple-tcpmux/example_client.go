// TCPMux 客户端示例
package main

import (
	"io"
	"log"
	"os"
	"time"
	"tcpmux"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("用法: %s <server_addr>", os.Args[0])
	}

	serverAddr := os.Args[1]

	// 连接到服务器
	client, err := tcpmux.Dial(serverAddr)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	log.Printf("已连接到服务器: %s", serverAddr)

	// 打开多个流进行测试
	for i := 0; i < 3; i++ {
		go func(id int) {
			stream, err := client.OpenStream()
			if err != nil {
				log.Printf("打开流失败: %v", err)
				return
			}
			defer stream.Close()

			log.Printf("流 %d 已打开: StreamID=%d", id, stream.ID())

			// 发送数据
			message := []byte("Hello from stream ")
			message = append(message, byte('0'+id))
			message = append(message, '\n')

			if _, err := stream.Write(message); err != nil {
				log.Printf("写入失败: %v", err)
				return
			}

			// 读取响应
			buf := make([]byte, 1024)
			n, err := stream.Read(buf)
			if err != nil && err != io.EOF {
				log.Printf("读取失败: %v", err)
				return
			}

			log.Printf("流 %d 收到响应: %s", id, string(buf[:n]))
		}(i)
	}

	// 等待一段时间
	time.Sleep(5 * time.Second)
	log.Println("测试完成")
}
