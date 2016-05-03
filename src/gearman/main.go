package main

import (
	"flag"
	"runtime"
	gearmand "server"
	"utils/logger"
)

const (
	version = "1.0.0.3"
)

var (
	addr     *string = flag.String("addr", ":4730", "listening on, such as :4730")
	monAddr  *string = flag.String("mon", ":5730", "listening on, such as :5730")
	logLevel *string = flag.String("verbose", "info", "log level, such as:trace info warn error")
	tryTimes *int    = flag.Int("trytime", 2, "wake worker try times if equal 0 wake all sleep worker")
	logPath  *string = flag.String("logpath", "./", "log path")
	maxProc  *int    = flag.Int("prosize", 1, " process size, if <=0 it is going to CPU num")
	lockMainProcess *bool = flag.Bool("lock", false, "lock EvtLoop process on specific cpu")
)

func main() {
	flag.Parse()

	procSize := 1
	cpuNum := runtime.NumCPU()

	if *maxProc <= 0 || *maxProc > cpuNum {
		procSize = cpuNum
	} else {
		procSize = *maxProc
	}

	runtime.GOMAXPROCS(procSize)
	logger.Initialize(*addr, *logLevel, *logPath)

	logger.Logger().I("gm server start up!!!! version:%v addr:%v mon:%v verbose:%v trytime:%v logpath:%v process size:%v lock:%v",
		version, *addr, *monAddr, *logLevel, *tryTimes, *logPath, procSize,
		*lockMainProcess)

	gearmand.NewServer(*tryTimes, procSize, *lockMainProcess).Start(*addr, *monAddr)
}
