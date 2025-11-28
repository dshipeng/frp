// 性能测试和对比
// 注意：这是一个独立的测试程序，需要单独运行
package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
	"tcpmux-wrapper"
)

func benchmarkMux(muxType tcpmux.MuxType, serverAddr string, streams int, duration time.Duration) {
	log.Printf("开始测试协议: %s，流数量: %d，持续时间: %v", muxType, streams, duration)

	// 连接到服务器
	client, err := tcpmux.Dial(serverAddr, muxType)
	if err != nil {
		log.Printf("连接失败: %v", err)
		return
	}
	defer client.Close()

	var wg sync.WaitGroup
	var totalBytes int64
	var totalRequests int64
	var mu sync.Mutex

	start := time.Now()
	end := start.Add(duration)

	// 创建多个流进行测试
	for i := 0; i < streams; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for time.Now().Before(end) {
				stream, err := client.OpenStream()
				if err != nil {
					log.Printf("打开流失败: %v", err)
					continue
				}

				// 发送数据
				data := []byte(fmt.Sprintf("test data from stream %d\n", id))
				n, err := stream.Write(data)
				if err != nil {
					stream.Close()
					continue
				}

				// 读取响应
				buf := make([]byte, 1024)
				m, err := stream.Read(buf)
				stream.Close()

				if err != nil && err != io.EOF {
					continue
				}

				mu.Lock()
				totalBytes += int64(n + m)
				totalRequests++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	mu.Lock()
	bytes := totalBytes
	requests := totalRequests
	mu.Unlock()

	log.Printf("协议 %s 测试结果:", muxType)
	log.Printf("  总请求数: %d", requests)
	log.Printf("  总字节数: %d", bytes)
	log.Printf("  耗时: %v", elapsed)
	log.Printf("  QPS: %.2f", float64(requests)/elapsed.Seconds())
	log.Printf("  吞吐量: %.2f KB/s", float64(bytes)/1024/elapsed.Seconds())
}

func main() {
	serverAddr := "localhost:8080"
	streams := 10
	duration := 10 * time.Second

	log.Println("开始性能测试...")
	log.Println("请确保服务器已启动")

	// 测试 yamux
	fmt.Println("\n" + "="*50)
	benchmarkMux(tcpmux.MuxTypeYamux, serverAddr, streams, duration)

	time.Sleep(2 * time.Second)

	// 测试 smux
	fmt.Println("\n" + "="*50)
	benchmarkMux(tcpmux.MuxTypeSmux, serverAddr, streams, duration)

	log.Println("\n性能测试完成")
}
