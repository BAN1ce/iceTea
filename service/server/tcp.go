package server

import (
	"log"
	"net"
	"strconv"
)

type TCPListener struct {
	listener *net.TCPListener
	addr     *net.TCPAddr
}

func NewTCPListener(host string, port int) *TCPListener {
	addr, err := net.ResolveTCPAddr("tcp", host+":"+strconv.Itoa(port))
	if err != nil {
		log.Fatalln(err)
	}
	return &TCPListener{
		addr: addr,
	}
}

func (t *TCPListener) Listen() error {
	var (
		err error
	)
	t.listener, err = net.ListenTCP("tcp", t.addr)
	return err
}

func (t *TCPListener) Accept() (net.Conn, error) {
	return t.listener.Accept()
}

func (t *TCPListener) Close() error {
	return t.listener.Close()
}

func (t *TCPListener) Addr() net.Addr {
	return t.listener.Addr()
}
