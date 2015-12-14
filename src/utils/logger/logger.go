package logger

import (
	"github.com/aiwuTech/fileLogger"
)

var (
	LOGGER *fileLogger.FileLogger
)

func init() {
}

func Initialize(prefix string, logLevel string) {
	LOGGER = fileLogger.NewDailyLogger("./log", "gearman_"+prefix+".log", "", 300, 5000)

	switch logLevel {
	case "trace":
		LOGGER.SetLogLevel(fileLogger.TRACE)
		break
	case "info":
		LOGGER.SetLogLevel(fileLogger.INFO)
		break
	case "warn":
		LOGGER.SetLogLevel(fileLogger.WARN)
		break
	case "error":
		LOGGER.SetLogLevel(fileLogger.ERROR)
		break
	default:
		LOGGER.SetLogLevel(fileLogger.INFO)
	}
}

func Close() {
	LOGGER.Close()
}

func Logger() *fileLogger.FileLogger {
	return LOGGER
}
