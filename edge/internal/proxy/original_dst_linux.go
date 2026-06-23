//go:build linux

/*
original_dst_linux.go reads the original destination from iptables-redirected
connections using the SO_ORIGINAL_DST socket option (Linux-only).
*/
package proxy

import (
	"fmt"
	"net"
	"strings"
	"syscall"
	"unsafe"
)

const (
	soOriginalDst = 80
)

func getOriginalDst(conn net.Conn) (string, int, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return "", 0, fmt.Errorf("not a TCP connection")
	}

	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return "", 0, fmt.Errorf("get syscall conn: %w", err)
	}

	var origAddr syscall.RawSockaddrInet4
	var origErr error

	err = rawConn.Control(func(fd uintptr) {
		addrLen := uint32(unsafe.Sizeof(origAddr))
		origErr = getsockopt(int(fd), syscall.SOL_IP, soOriginalDst, unsafe.Pointer(&origAddr), &addrLen)
	})
	if err != nil {
		return "", 0, fmt.Errorf("control fd: %w", err)
	}
	if origErr != nil {
		return "", 0, fmt.Errorf("SO_ORIGINAL_DST: %w", origErr)
	}

	ip := net.IPv4(origAddr.Addr[0], origAddr.Addr[1], origAddr.Addr[2], origAddr.Addr[3])
	port := int(origAddr.Port>>8) | int(origAddr.Port<<8&0xff00)

	// Try reverse DNS, fall back to IP
	names, err := net.LookupAddr(ip.String())
	host := ip.String()
	if err == nil && len(names) > 0 {
		host = strings.TrimSuffix(names[0], ".")
	}

	return host, port, nil
}

func getsockopt(fd int, level, name int, val unsafe.Pointer, vallen *uint32) error {
	_, _, errno := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		uintptr(fd),
		uintptr(level),
		uintptr(name),
		uintptr(val),
		uintptr(unsafe.Pointer(vallen)),
		0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}
