package controllermanager

import (
	"fmt"
	"net"

	netutils "k8s.io/utils/net"
)

// AppendPortIfNeeded 将给定端口追加到 IP 地址，除非它已经是
// "ipv4:port" 或 "[ipv6]:port" 格式。
func AppendPortIfNeeded(addr string, port int32) string {
	// 如果地址已经是 "ipv4:port" 或 "[ipv6]:port" 格式，则直接返回。
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}

	// 对于无效情况直接返回。这应该由验证来捕获。
	ip := netutils.ParseIPSloppy(addr)
	if ip == nil {
		return addr
	}

	// 将端口追加到地址。
	if ip.To4() != nil {
		return fmt.Sprintf("%s:%d", addr, port)
	}
	return fmt.Sprintf("[%s]:%d", addr, port)
}
