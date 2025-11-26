#!/bin/bash

# 代理服务测试脚本

FRPS_HOST="${FRPS_HOST:-127.0.0.1}"
FRPS_PORT="${FRPS_PORT:-8888}"
PROXY_URL="http://${FRPS_HOST}:${FRPS_PORT}"

echo "========================================="
echo "测试代理服务"
echo "========================================="
echo "代理地址: $PROXY_URL"
echo ""

# 测试 HTTP 请求
echo "1. 测试 HTTP 请求..."
echo "   请求: http://httpbin.org/get"
curl -x "$PROXY_URL" -s http://httpbin.org/get | head -20
echo ""

# 测试 HTTPS 请求
echo "2. 测试 HTTPS 请求..."
echo "   请求: https://httpbin.org/get"
curl -x "$PROXY_URL" -s https://httpbin.org/get | head -20
echo ""

# 测试获取 IP
echo "3. 测试获取客户端 IP..."
echo "   请求: http://httpbin.org/ip"
curl -x "$PROXY_URL" -s http://httpbin.org/ip
echo ""

# 测试 POST 请求
echo "4. 测试 POST 请求..."
echo "   请求: http://httpbin.org/post"
curl -x "$PROXY_URL" -X POST -H "Content-Type: application/json" \
     -d '{"test":"data"}' -s http://httpbin.org/post | head -20
echo ""

echo "========================================="
echo "测试完成"
echo "========================================="
