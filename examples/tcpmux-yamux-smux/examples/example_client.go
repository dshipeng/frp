// TCPMux 客户端示例
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
	"tcpmux-wrapper"
)

func main() {
	var (
		serverAddr = flag.String("addr", "localhost:8080", "服务器地址")
		muxType    = flag.String("mux", "yamux", "多路复用协议类型 (yamux/smux)")
		streams    = flag.Int("streams", 3, "创建的流数量")
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

	// 连接到服务器
	client, err := tcpmux.Dial(*serverAddr, mux)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	log.Printf("已连接到服务器: %s (协议: %s)", *serverAddr, mux)

	var wg sync.WaitGroup

	// 打开多个流进行测试
	for i := 0; i < *streams; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			stream, err := client.OpenStream()
			if err != nil {
				log.Printf("打开流失败: %v", err)
				return
			}
			defer stream.Close()

			log.Printf("流 %d 已打开: %s -> %s", id, stream.LocalAddr(), stream.RemoteAddr())

			// 发送数据
			message := fmt.Sprintf("Hello from stream %d\n", id)
			if _, err := stream.Write([]byte(message)); err != nil {
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

	// 等待所有流完成
	wg.Wait()
	time.Sleep(1 * time.Second)
	log.Println("测试完成")
}
