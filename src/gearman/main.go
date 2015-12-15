package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	gearmand "server"
	"syscall"
	"utils/logger"
)

var (
	addr      *string = flag.String("addr", ":4730", "listening on, such as :4730")
	monAddr   *string = flag.String("mon", ":1374", "listening on, such as :1374")
	logLevel  *string = flag.String("verbose", "info", "log level, such as:trace info warn error")
	tryTimes  *int    = flag.Int("trytime", 2, "wake worker try times if equal 0 wake all sleep worker")
	isDaemon  *bool   = flag.Bool("d", false, "make process daemon")
	keepAlive *int64  = flag.Int64("keepalive", 3, "keepalive Minute")
)

func daemon(nochdir, noclose int) int {
	var ret, ret2 uintptr
	var err syscall.Errno

	darwin := runtime.GOOS == "darwin"

	// already a daemon
	if syscall.Getppid() == 1 {
		return 0
	}

	// fork off the parent process
	ret, ret2, err = syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)
	if err != 0 {
		return -1
	}

	// failure
	if ret2 < 0 {
		os.Exit(-1)
	}

	// handle exception for darwin
	if darwin && ret2 == 1 {
		ret = 0
	}

	// if we got a good PID, then we call exit the parent process.
	if ret > 0 {
		os.Exit(0)
	}

	/* Change the file mode mask */
	_ = syscall.Umask(0)

	// create a new SID for the child process
	s_ret, s_errno := syscall.Setsid()
	if s_errno != nil {
		log.Printf("Error: syscall.Setsid errno: %d", s_errno)
	}
	if s_ret < 0 {
		return -1
	}

	if nochdir == 0 {
		os.Chdir("/")
	}

	if noclose == 0 {
		f, e := os.OpenFile("/dev/null", os.O_RDWR, 0)
		if e == nil {
			fd := f.Fd()
			syscall.Dup2(int(fd), int(os.Stdin.Fd()))
			syscall.Dup2(int(fd), int(os.Stdout.Fd()))
			syscall.Dup2(int(fd), int(os.Stderr.Fd()))
		}
	}

	return 0
}

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(1)
	logger.Initialize(*addr, *logLevel)

	logger.Logger().I("gm server start up!!!! addr:%v mon:%v verbose:%v trytime:%v daemon:%v keepalive:%v",
		*addr, *monAddr, *logLevel, *tryTimes, *isDaemon, *keepAlive)

	if *isDaemon {
		daemon(1, 0)
	}

	gearmand.NewServer(*tryTimes, *keepAlive).Start(*addr, *monAddr)
}
