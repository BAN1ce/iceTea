package server

import (
	"net"
)

type Listener interface {
	Accept() (net.Conn, error)
	Close() error
	Listen() error
}
