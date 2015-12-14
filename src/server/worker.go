package server

import (
	// "bytes"
	// . "common"
	// "encoding/json"
	"net"
)

const (
	wsRunning         = 1
	wsSleep           = 2
	wsPrepareForSleep = 3
)

func status2str(status int) string {
	switch status {
	case wsRunning:
		return "running"
	case wsSleep:
		return "sleep"
	case wsPrepareForSleep:
		return "prepareForSleep"
	}

	return "unknown"
}

type Worker struct {
	Conn net.Conn
	Connector

	workerId string
	status   int
	canDo    map[string]bool
}
