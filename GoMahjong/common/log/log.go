package log

import (
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

var logger *log.Logger

func InitLog(appName string, logLevel string) {
	logger = log.New(os.Stderr)
	logger.SetPrefix(appName)
	logger.SetReportTimestamp(true)
	logger.SetTimeFormat(time.DateTime)

	// 默认为 info 级别
	if logLevel == "" {
		logLevel = "info"
	}

	logLevel = strings.ToLower(logLevel)
	switch logLevel {
	case "debug":
		logger.SetLevel(log.DebugLevel)
	case "warn":
		logger.SetLevel(log.WarnLevel)
	case "error":
		logger.SetLevel(log.ErrorLevel)
	default:
		logger.SetLevel(log.InfoLevel)
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
