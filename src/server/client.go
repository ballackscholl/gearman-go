package server

import (
	"net"
	"time"
)

type Connector struct {
	SessionId int64
	in        chan []byte
	ConnectAt time.Time
}

func (connector *Connector) Send(data []byte) {
	connector.in <- data
}

type Client struct {
	Conn net.Conn
	Connector
}
