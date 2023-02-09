package service

import (
	"icetea/client"
	"sync"
)

var (
	DefaultClientService = NewClientService()
)

type ConnectionService struct {
	mux         sync.RWMutex
	connections map[string]*client.Client
}

func (l *ConnectionService) CreateClient(client *client.Client) {
	l.mux.Lock()
	defer l.mux.Unlock()
	if c, ok := l.connections[client.GetConn().LocalAddr().String()]; ok {
		c.Close()
	}
	l.connections[client.GetId()] = client
}

func NewClientService() *ConnectionService {
	return &ConnectionService{connections: make(map[string]*client.Client)}
}
