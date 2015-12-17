package main

import (
	"flag"
	"runtime"
	gearmand "server"
	"utils/logger"
)

var (
	addr      *string = flag.String("addr", ":4730", "listening on, such as :4730")
	monAddr   *string = flag.String("mon", ":1374", "listening on, such as :1374")
	logLevel  *string = flag.String("verbose", "info", "log level, such as:trace info warn error")
	tryTimes  *int    = flag.Int("trytime", 2, "wake worker try times if equal 0 wake all sleep worker")
	keepAlive *int64  = flag.Int64("keepalive", 3, "keepalive Minute")
)

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(1)
	logger.Initialize(*addr, *logLevel)

	logger.Logger().I("gm server start up!!!! addr:%v mon:%v verbose:%v trytime:%v keepalive:%v",
		*addr, *monAddr, *logLevel, *tryTimes, *keepAlive)

	gearmand.NewServer(*tryTimes, *keepAlive).Start(*addr, *monAddr)
}
