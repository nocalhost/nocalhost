package tun

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/songgao/water"
	"net"
	"os"
	"time"
)

// Config is the config for TUN device.
type Config struct {
	Name    string
	Addr    string
	MTU     int
	Routes  []IPRoute
	Gateway string
}

type tunListener struct {
	addr   net.Addr
	conns  chan net.Conn
	closed chan struct{}
	config Config
}

// Listener TunListener creates a listener for tun tunnel.
func Listener(config Config) (net.Listener, error) {
	ln := &tunListener{
		conns:  make(chan net.Conn, 1),
		closed: make(chan struct{}),
		config: config,
	}

	conn, ifce, err := createTun(config)
	if err != nil {
		return nil, err
	}
	ln.addr = conn.LocalAddr()

	addrs, _ := ifce.Addrs()
	_ = os.Setenv("tunName", ifce.Name)
	log.Debugf("[tun] %s: name: %s, mtu: %d, addrs: %s", conn.LocalAddr(), ifce.Name, ifce.MTU, addrs)

	ln.conns <- conn

	return ln, nil
}

func (l *tunListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.conns:
		return conn, nil
	case <-l.closed:
	}

	return nil, errors.New("accept on closed listener")
}

func (l *tunListener) Addr() net.Addr {
	return l.addr
}

func (l *tunListener) Close() error {
	select {
	case <-l.closed:
		return errors.New("listener has been closed")
	default:
		close(l.closed)
	}
	return nil
}

type tunConn struct {
	ifce *water.Interface
	addr net.Addr
}

func (c *tunConn) Read(b []byte) (n int, err error) {
	return c.ifce.Read(b)
}

func (c *tunConn) Write(b []byte) (n int, err error) {
	return c.ifce.Write(b)
}

func (c *tunConn) Close() (err error) {
	return c.ifce.Close()
}

func (c *tunConn) LocalAddr() net.Addr {
	return c.addr
}

func (c *tunConn) RemoteAddr() net.Addr {
	return &net.IPAddr{}
}

func (c *tunConn) SetDeadline(time.Time) error {
	return &net.OpError{Op: "set", Net: "tun", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *tunConn) SetReadDeadline(time.Time) error {
	return &net.OpError{Op: "set", Net: "tun", Source: nil, Addr: nil, Err: errors.New("read deadline not supported")}
}

func (c *tunConn) SetWriteDeadline(time.Time) error {
	return &net.OpError{Op: "set", Net: "tun", Source: nil, Addr: nil, Err: errors.New("write deadline not supported")}
}

// IPRoute is an IP routing entry.
type IPRoute struct {
	Dest    *net.IPNet
	Gateway net.IP
}
