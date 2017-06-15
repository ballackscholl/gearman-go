package server

import (
	"net"
	"time"
	"sync"
)

type Connector struct {
	SessionId int64
	in        chan []byte
	ConnectAt time.Time
	isConnect bool
	locker 	sync.Mutex
}

func (connector *Connector) SetIsConnect(isConnect bool) {
	connector.isConnect = isConnect;
}

func (connector *Connector) IsConnect() bool {
	return connector.isConnect;
}

func (connector *Connector) Send(data []byte) {
	connector.locker.Lock()
	if connector.isConnect{
		connector.in <- data
	}
	connector.locker.Unlock()
}

type Client struct {
	Conn net.Conn
	Connector
}
