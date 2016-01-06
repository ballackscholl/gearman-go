package main

import (
	"flag"
	"runtime"
	gearmand "server"
	"utils/logger"
)

var (
	addr     *string = flag.String("addr", ":4730", "listening on, such as :4730")
	monAddr  *string = flag.String("mon", ":5730", "listening on, such as :5730")
	logLevel *string = flag.String("verbose", "info", "log level, such as:trace info warn error")
	tryTimes *int    = flag.Int("trytime", 2, "wake worker try times if equal 0 wake all sleep worker")
	logPath  *string = flag.String("logpath", "./", "log path default ./")
)

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(1)
	logger.Initialize(*addr, *logLevel, *logPath)

	logger.Logger().I("gm server start up!!!! addr:%v mon:%v verbose:%v trytime:%v logpath:%v",
		*addr, *monAddr, *logLevel, *tryTimes, *logPath)

	gearmand.NewServer(*tryTimes).Start(*addr, *monAddr)
}
