package util

import (
	"bytes"
	"net"
	"sync"
	"time"
)

// Debug is a flag that enables the debug log.
var Debug bool

var (
	smallBufferSize  = 2 * 1024  // 2KB small buffer
	mediumBufferSize = 8 * 1024  // 8KB medium buffer
	largeBufferSize  = 32 * 1024 // 32KB large buffer
)

var (
	SPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, smallBufferSize)
		},
	}
	MPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, mediumBufferSize)
		},
	}
	LPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, largeBufferSize)
		},
	}
)

var (
	// KeepAliveTime is the keep alive time period for TCP connection.
	KeepAliveTime = 180 * time.Second
	// DialTimeout is the timeout of dial.
	DialTimeout = 15 * time.Second
	// HandshakeTimeout is the timeout of handshake.
	HandshakeTimeout = 5 * time.Second
	// ConnectTimeout is the timeout for connect.
	ConnectTimeout = 5 * time.Second
	// ReadTimeout is the timeout for reading.
	ReadTimeout = 10 * time.Second
	// WriteTimeout is the timeout for writing.
	WriteTimeout = 10 * time.Second
)

var (
	DefaultMTU = getMTU()
)

func getMTU() int {
	if ift, err := net.Interfaces(); err == nil {
		for _, ifi := range ift {
			if ifi.Flags&net.FlagUp != 0 && bytes.Compare(ifi.HardwareAddr, nil) != 0 {
				return ifi.MTU
			}
		}
	}
	return 1350
}
