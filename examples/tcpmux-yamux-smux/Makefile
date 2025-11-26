.PHONY: build server client test clean deps help

help:
	@echo "可用的命令:"
	@echo "  make deps    - 下载依赖"
	@echo "  make build   - 编译所有程序"
	@echo "  make server  - 编译服务端"
	@echo "  make client  - 编译客户端"
	@echo "  make test    - 运行测试"
	@echo "  make clean   - 清理编译产物"

deps:
	@echo "下载依赖..."
	go mod download
	go mod tidy

build: server client

server:
	@echo "编译服务端..."
	cd examples && go build -o ../server example_server.go

client:
	@echo "编译客户端..."
	cd examples && go build -o ../client example_client.go

test: build
	@echo "启动服务端（后台，使用 yamux）..."
	./server -mux yamux &
	@sleep 2
	@echo "运行客户端测试..."
	./client -mux yamux -streams 3
	@pkill -f "./server" || true
	@sleep 1
	@echo "启动服务端（后台，使用 smux）..."
	./server -mux smux &
	@sleep 2
	@echo "运行客户端测试..."
	./client -mux smux -streams 3
	@pkill -f "./server" || true

clean:
	@echo "清理编译产物..."
	rm -f server client
	@echo "清理完成"
