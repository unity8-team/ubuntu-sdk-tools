package shared

import (
	"fmt"
	log "gopkg.in/inconshreveable/log15.v2"
	"runtime"
)

type Logger interface {
	Debug(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
	Crit(msg string, ctx ...interface{})
}

var Log Logger

type nullLogger struct{}

func (nl nullLogger) Debug(msg string, ctx ...interface{}) {}
func (nl nullLogger) Info(msg string, ctx ...interface{})  {}
func (nl nullLogger) Warn(msg string, ctx ...interface{})  {}
func (nl nullLogger) Error(msg string, ctx ...interface{}) {}
func (nl nullLogger) Crit(msg string, ctx ...interface{})  {}

func init() {
	Log = nullLogger{}
}

// General wrappers around Logger interface functions.
func LogDebug(msg string, ctx map[string]interface{}) {
	if Log != nil {
		Log.Debug(msg, log.Ctx(ctx))
	}
}

func LogInfo(msg string, ctx map[string]interface{}) {
	if Log != nil {
		Log.Info(msg, log.Ctx(ctx))
	}
}

func LogWarn(msg string, ctx map[string]interface{}) {
	if Log != nil {
		Log.Warn(msg, log.Ctx(ctx))
	}
}

func LogError(msg string, ctx map[string]interface{}) {
	if Log != nil {
		Log.Error(msg, log.Ctx(ctx))
	}
}

func LogCrit(msg string, ctx map[string]interface{}) {
	if Log != nil {
		Log.Crit(msg, log.Ctx(ctx))
	}
}

// Wrappers around Logger interface functions that send a string to the Logger
// by running it through fmt.Sprintf().
func LogInfof(format string, args ...interface{}) {
	if Log != nil {
		Log.Info(fmt.Sprintf(format, args...))
	}
}

func LogDebugf(format string, args ...interface{}) {
	if Log != nil {
		Log.Debug(fmt.Sprintf(format, args...))
	}
}

func LogWarnf(format string, args ...interface{}) {
	if Log != nil {
		Log.Warn(fmt.Sprintf(format, args...))
	}
}

func LogErrorf(format string, args ...interface{}) {
	if Log != nil {
		Log.Error(fmt.Sprintf(format, args...))
	}
}

func LogCritf(format string, args ...interface{}) {
	if Log != nil {
		Log.Crit(fmt.Sprintf(format, args...))
	}
}

func PrintStack() {
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, true)
	LogDebugf("%s", buf)
}
