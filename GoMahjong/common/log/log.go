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
	} else if config.Conf.Log.Level == "debug" {
		logger.SetLevel(log.DebugLevel)
	} else {
		logger.SetLevel(log.WarnLevel)
	}
}

func Fatal(format string, args ...any) {
	if len(args) == 0 {
		logger.Fatalf(format)
	} else {
		logger.Fatalf(format, args...)
	}
}

func Info(format string, args ...any) {
	if len(args) == 0 {
		logger.Infof(format)
	} else {
		logger.Infof(format, args...)
	}
}

func Warn(format string, args ...any) {
	if len(args) == 0 {
		logger.Warnf(format)
	} else {
		logger.Warnf(format, args...)
	}
}

func Error(format string, args ...any) {
	if len(args) == 0 {
		logger.Errorf(format)
	} else {
		logger.Errorf(format, args...)
	}
}

func Debug(format string, args ...any) {
	if len(args) == 0 {
		logger.Debugf(format)
	} else {
		logger.Debugf(format, args...)
	}
}
