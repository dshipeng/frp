#!/bin/bash

# 启动脚本示例

echo "========================================="
echo "启动基于 frp TCPMux 的代理服务"
echo "========================================="

# 检查配置文件是否存在
if [ ! -f "frps.toml" ]; then
    echo "错误: frps.toml 不存在"
    exit 1
fi

if [ ! -f "frpc.toml" ]; then
    echo "错误: frpc.toml 不存在"
    exit 1
fi

# 检查代理服务器程序是否存在
if [ ! -f "proxy_agent" ]; then
    echo "编译代理服务器..."
    go build -o proxy_agent proxy_agent.go
    if [ $? -ne 0 ]; then
        echo "编译失败"
        exit 1
    fi
fi

echo ""
echo "1. 启动代理服务器 (proxy_agent)..."
./proxy_agent &
PROXY_PID=$!
echo "   代理服务器 PID: $PROXY_PID"

sleep 2

echo ""
echo "2. 启动 frpc..."
echo "   请确保 frps 已经在服务端启动"
echo "   按 Ctrl+C 停止所有服务"
echo ""

# 启动 frpc（前台运行，方便查看日志）
frpc -c frpc.toml &
FRPC_PID=$!

# 等待用户中断
trap "echo ''; echo '正在停止服务...'; kill $PROXY_PID $FRPC_PID 2>/dev/null; exit" INT TERM

wait
