.PHONY: build run test clean help

# 默认目标
help:
	@echo "可用的命令:"
	@echo "  make build    - 编译代理服务器"
	@echo "  make run      - 运行代理服务器"
	@echo "  make test     - 测试代理服务"
	@echo "  make clean    - 清理编译产物"
	@echo ""
	@echo "使用示例:"
	@echo "  make build && make run"

# 编译代理服务器
build:
	@echo "编译代理服务器..."
	go build -o proxy_agent proxy_agent.go
	@echo "编译完成: proxy_agent"

# 运行代理服务器
run: build
	@echo "启动代理服务器..."
	./proxy_agent

# 测试代理服务
test:
	@echo "测试代理服务..."
	@if [ -z "$$FRPS_HOST" ]; then \
		echo "提示: 设置 FRPS_HOST 环境变量指定服务端地址"; \
		echo "例如: FRPS_HOST=your_server make test"; \
	fi
	./test_proxy.sh

# 清理
clean:
	@echo "清理编译产物..."
	rm -f proxy_agent proxy_agent.exe
	@echo "清理完成"
