package log

import (
	"fmt"
	"github.com/mattn/go-colorable"
	"github.com/zdarovich/fibbage-game-server/internal/config"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/snowzach/rotatefilehook"
)

var (
	ip   string
	host string
	pid  string
)

//New init ...
func init() {
	configuration := &config.Configuration.Logger

	if configuration == nil {
		return
	}

	var logLevel logrus.Level
	if configuration.LogDebugMode {
		logLevel = logrus.DebugLevel
	} else {
		logLevel = logrus.InfoLevel
	}
	if configuration.LogsFileEnabled {
		rotateFileHook, err := rotatefilehook.NewRotateFileHook(rotatefilehook.RotateFileConfig{
			Filename:   fmt.Sprintf("%s.%s.%s.%s", configuration.LogFilePath, configuration.LogFileName, logLevel.String(), time.Now().Format("01-02-2006")),
			MaxSize:    configuration.LogRotateMegabytes, // megabytes
			MaxBackups: configuration.LogRotateFiles,
			MaxAge:     configuration.LogRotateDuration, //days
			Level:      logLevel,
			Formatter: &logrus.JSONFormatter{
				TimestampFormat: time.RFC822,
			},
		})
		if err != nil {
			logrus.Fatalf("Failed to initialize file rotate hook: %v", err)
		} else {
			logrus.AddHook(rotateFileHook)
		}

	} else {
		logrus.SetOutput(colorable.NewColorableStdout())

		logrus.SetLevel(logLevel)
	}

	var ipErr error

	ip, ipErr = externalIP()
	if ipErr != nil {
		ip = ipErr.Error()
	}
	hosts, err := net.LookupAddr(ip)
	if err != nil {
		host = err.Error()
	}
	host = strings.Join(hosts, " ")

	pid = fmt.Sprint(os.Getpid())

}

//Trace wrapper
func Trace(args ...interface{}) {
	withContext().Trace(args...)
}

//Debug wrapper
func Debug(args ...interface{}) {
	withContext().Debug(args...)
}

//Debug wrapper
func Debugf(format string, args ...interface{}) {
	withContext().Debugf(format, args...)
}

//Info wrapper
func Info(args ...interface{}) {
	withContext().Info(args...)
}

//Infof wrapper
func Infof(format string, args ...interface{}) {
	withContext().Infof(format, args...)
}

//Warn wrapper
func Warn(args ...interface{}) {
	withContext().Warn(args...)
}

//Error wrapper
func Error(args ...interface{}) {
	withContext().Error(args...)
}

//Errorf wrapper
func Errorf(format string, args ...interface{}) {
	withContext().Errorf(format, args...)
}

//Fatal wrapper
func Fatal(args ...interface{}) {
	withContext().Fatal(args...)
}

//Panic wrapper
func Panic(args ...interface{}) {
	withContext().Panic(args...)
}

//HTTP ...
func HTTP(req *http.Request, res *http.Response, err error, duration time.Duration) {

	withContext().Infof(
		"Request %s %s",
		req.Method,
		req.URL.String(),
	)
	if err != nil {
		logrus.Error(err)
		return
	}

	duration /= time.Millisecond
	withContext().Infof(
		"Response status=%d durationMs=%d",
		res.StatusCode,
		duration,
	)
}
