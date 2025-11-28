// SOCKS5 代理协议处理（用于 TCPMux 流）
package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

// SOCKS5 协议常量
const (
	SOCKS5Version = 0x05
	SOCKS5CmdConnect = 0x01
	SOCKS5CmdBind = 0x02
	SOCKS5CmdUDP = 0x03
	
	SOCKS5ATYPIPv4 = 0x01
	SOCKS5ATYPIPv6 = 0x04
	SOCKS5ATYPDomain = 0x03
	
	SOCKS5RepSuccess = 0x00
	SOCKS5RepFailure = 0x01
)

// SOCKS5Request SOCKS5 请求结构
type SOCKS5Request struct {
	Version byte
	Command byte
	ATYP    byte
	Address string
	Port    uint16
}

// SendSOCKS5Request 通过 TCPMux 流发送 SOCKS5 请求
func SendSOCKS5Request(stream net.Conn, targetHost string, targetPort string) error {
	// 解析端口
	var port uint16
	if _, err := fmt.Sscanf(targetPort, "%d", &port); err != nil {
		return fmt.Errorf("invalid port: %s", targetPort)
	}

	// 构建 SOCKS5 请求
	req := &SOCKS5Request{
		Version: SOCKS5Version,
		Command: SOCKS5CmdConnect,
		Address: targetHost,
		Port:    port,
	}

	// 确定地址类型
	var atyp byte
	var addrBytes []byte
	
	ip := net.ParseIP(targetHost)
	if ip != nil {
		if ip.To4() != nil {
			// IPv4
			atyp = SOCKS5ATYPIPv4
			addrBytes = ip.To4()
		} else {
			// IPv6
			atyp = SOCKS5ATYPIPv6
			addrBytes = ip.To16()
		}
	} else {
		// 域名
		atyp = SOCKS5ATYPDomain
		addrBytes = []byte(targetHost)
	}

	// 构建请求包
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	buf := make([]byte, 0, 6+len(addrBytes))
	buf = append(buf, SOCKS5Version) // VER
	buf = append(buf, SOCKS5CmdConnect) // CMD
	buf = append(buf, 0x00) // RSV
	buf = append(buf, atyp) // ATYP
	
	if atyp == SOCKS5ATYPDomain {
		// 域名长度 + 域名
		buf = append(buf, byte(len(addrBytes)))
	}
	buf = append(buf, addrBytes...) // DST.ADDR
	
	// 端口（2字节，大端序）
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, port)
	buf = append(buf, portBytes...) // DST.PORT

	// 发送请求
	_, err := stream.Write(buf)
	return err
}

// ReceiveSOCKS5Request 从 TCPMux 流接收 SOCKS5 请求（客户端使用）
func ReceiveSOCKS5Request(stream net.Conn) (*SOCKS5Request, error) {
	// 读取版本和命令（至少4字节）
	buf := make([]byte, 4)
	if _, err := stream.Read(buf); err != nil {
		return nil, fmt.Errorf("failed to read SOCKS5 header: %v", err)
	}

	version := buf[0]
	command := buf[1]
	_ = buf[2] // RSV
	atyp := buf[3]

	if version != SOCKS5Version {
		return nil, fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	// 读取地址
	var address string
	var addrBytes []byte

	switch atyp {
	case SOCKS5ATYPIPv4:
		// IPv4: 4字节
		addrBytes = make([]byte, 4)
		if _, err := stream.Read(addrBytes); err != nil {
			return nil, fmt.Errorf("failed to read IPv4 address: %v", err)
		}
		address = net.IP(addrBytes).String()

	case SOCKS5ATYPIPv6:
		// IPv6: 16字节
		addrBytes = make([]byte, 16)
		if _, err := stream.Read(addrBytes); err != nil {
			return nil, fmt.Errorf("failed to read IPv6 address: %v", err)
		}
		address = net.IP(addrBytes).String()

	case SOCKS5ATYPDomain:
		// 域名: 1字节长度 + 域名
		lenBuf := make([]byte, 1)
		if _, err := stream.Read(lenBuf); err != nil {
			return nil, fmt.Errorf("failed to read domain length: %v", err)
		}
		domainLen := int(lenBuf[0])
		addrBytes = make([]byte, domainLen)
		if _, err := stream.Read(addrBytes); err != nil {
			return nil, fmt.Errorf("failed to read domain: %v", err)
		}
		address = string(addrBytes)

	default:
		return nil, fmt.Errorf("unsupported address type: %d", atyp)
	}

	// 读取端口（2字节，大端序）
	portBytes := make([]byte, 2)
	if _, err := stream.Read(portBytes); err != nil {
		return nil, fmt.Errorf("failed to read port: %v", err)
	}
	port := binary.BigEndian.Uint16(portBytes)

	return &SOCKS5Request{
		Version: version,
		Command: command,
		ATYP:    atyp,
		Address: address,
		Port:    port,
	}, nil
}

// SOCKS5Response SOCKS5 响应结构
type SOCKS5Response struct {
	Version byte
	Rep     byte // 响应码（0x00=成功）
	ATYP    byte
	Address string
	Port    uint16
}

// ReceiveSOCKS5Response 接收 SOCKS5 响应（服务端使用）
func ReceiveSOCKS5Response(stream net.Conn) (*SOCKS5Response, error) {
	// 读取版本和响应码（至少4字节）
	buf := make([]byte, 4)
	if _, err := stream.Read(buf); err != nil {
		return nil, fmt.Errorf("failed to read SOCKS5 response header: %v", err)
	}

	version := buf[0]
	rep := buf[1] // 响应码
	_ = buf[2]    // RSV
	atyp := buf[3]

	if version != SOCKS5Version {
		return nil, fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	// 读取地址（与请求格式相同）
	var address string

	switch atyp {
	case SOCKS5ATYPIPv4:
		addrBytes := make([]byte, 4)
		if _, err := stream.Read(addrBytes); err != nil {
			return nil, fmt.Errorf("failed to read IPv4 address: %v", err)
		}
		address = net.IP(addrBytes).String()

	case SOCKS5ATYPIPv6:
		addrBytes := make([]byte, 16)
		if _, err := stream.Read(addrBytes); err != nil {
			return nil, fmt.Errorf("failed to read IPv6 address: %v", err)
		}
		address = net.IP(addrBytes).String()

	case SOCKS5ATYPDomain:
		lenBuf := make([]byte, 1)
		if _, err := stream.Read(lenBuf); err != nil {
			return nil, fmt.Errorf("failed to read domain length: %v", err)
		}
		domainLen := int(lenBuf[0])
		addrBytes := make([]byte, domainLen)
		if _, err := stream.Read(addrBytes); err != nil {
			return nil, fmt.Errorf("failed to read domain: %v", err)
		}
		address = string(addrBytes)

	default:
		return nil, fmt.Errorf("unsupported address type: %d", atyp)
	}

	// 读取端口（2字节，大端序）
	portBytes := make([]byte, 2)
	if _, err := stream.Read(portBytes); err != nil {
		return nil, fmt.Errorf("failed to read port: %v", err)
	}
	port := binary.BigEndian.Uint16(portBytes)

	return &SOCKS5Response{
		Version: version,
		Rep:     rep,
		ATYP:    atyp,
		Address: address,
		Port:    port,
	}, nil
}

// SendSOCKS5Response 发送 SOCKS5 响应（客户端使用）
func SendSOCKS5Response(stream net.Conn, rep byte, address string, port uint16) error {
	// 构建响应包
	// +----+-----+-------+------+----------+----------+
	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	
	buf := make([]byte, 0, 6)
	buf = append(buf, SOCKS5Version) // VER
	buf = append(buf, rep) // REP
	buf = append(buf, 0x00) // RSV
	buf = append(buf, SOCKS5ATYPIPv4) // ATYP (简化：总是返回 IPv4)
	
	// BND.ADDR: 0.0.0.0 (4字节)
	buf = append(buf, 0, 0, 0, 0)
	
	// BND.PORT: 0 (2字节)
	buf = append(buf, 0, 0)

	_, err := stream.Write(buf)
	return err
}
