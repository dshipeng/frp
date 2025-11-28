// 代理配置
package main

import (
	"encoding/json"
	"os"
)

// ProxyConfig 代理配置
type ProxyConfig struct {
	// 使用 IPv6 连接目标站点（默认 false，使用 IPv4）
	UseIPv6 bool `json:"useIPv6"`
	// 优先使用 IPv6，如果失败则回退到 IPv4
	PreferIPv6 bool `json:"preferIPv6"`
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*ProxyConfig, error) {
	if path == "" {
		// 默认配置
		return &ProxyConfig{
			UseIPv6:    false,
			PreferIPv6: false,
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config ProxyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(path string, config *ProxyConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
