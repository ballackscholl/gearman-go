package main

import (
	"flag"
	"runtime"
	gearmand "server"
	"utils/logger"
)

var (
	addr     *string = flag.String("addr", ":4730", "listening on, such as 0.0.0.0:4730")
	logLevel *string = flag.String("verbose", "info", "log level")
)

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(1)
	logger.Initialize(*addr, *logLevel)

	gearmand.NewServer().Start(*addr)
}
