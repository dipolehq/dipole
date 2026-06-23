//go:build !linux

/*
original_dst_other.go provides a stub for non-Linux platforms.
SO_ORIGINAL_DST is Linux-only. On other platforms, the transparent listener
cannot resolve original destinations and will fail-closed.
*/
package proxy

import (
	"fmt"
	"net"
)

func getOriginalDst(conn net.Conn) (string, int, error) {
	return "", 0, fmt.Errorf("SO_ORIGINAL_DST not supported on this platform (Linux only)")
}
