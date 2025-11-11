package log

import (
	"common/config"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

var logger *log.Logger

func InitLog(appName string) {
	logger = log.New(os.Stderr)
	logger.SetPrefix(appName)
	logger.SetReportTimestamp(true)
	logger.SetTimeFormat(time.DateTime)

	if config.Conf.Log.Level == "info" {
		logger.SetLevel(log.InfoLevel)
	}
}

func Fatal(format string, args ...any) {
	if len(args) == 0 {
		logger.Fatal(format)
	} else {
		logger.Fatal(format, args...)
	}
}

func Info(format string, args ...any) {
	if len(args) == 0 {
		logger.Info(format)
	} else {
		logger.Info(format, args...)
	}
}

func Warn(format string, args ...any) {
	if len(args) == 0 {
		logger.Warn(format)
	} else {
		logger.Warn(format, args...)
	}
}

func Error(format string, args ...any) {
	if len(args) == 0 {
		logger.Error(format)
	} else {
		logger.Error(format, args...)
	}
}

func Debug(format string, args ...any) {
	if len(args) == 0 {
		logger.Debug(format)
	} else {
		logger.Debug(format, args...)
	}
}
