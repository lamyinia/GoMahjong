package log

import (
	"os"
	"strings"
	"time"

	charmlog "github.com/charmbracelet/log"
)

var logger *charmlog.Logger

func InitLog(appName string, logLevel string) {
	logger = charmlog.New(os.Stdout)
	logger.SetPrefix(appName)
	logger.SetReportTimestamp(true)
	logger.SetTimeFormat(time.DateTime)

	logger.SetReportCaller(true)
	if logLevel == "" {
		logLevel = "info"
	}

	logLevel = strings.ToLower(logLevel)
	switch logLevel {
	case "debug":
		logger.SetLevel(charmlog.DebugLevel)
	case "warn":
		logger.SetLevel(charmlog.WarnLevel)
	case "error":
		logger.SetLevel(charmlog.ErrorLevel)
	default:
		logger.SetLevel(charmlog.InfoLevel)
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
