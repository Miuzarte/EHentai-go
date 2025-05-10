package log

import (
	"github.com/Miuzarte/SimpleLog"
)

var log = SimpleLog.New("[EH-cli]", true, false)

func Trace(a ...any) {
	log.Trace(a...)
}

func Tracef(format string, a ...any) {
	log.Tracef(format, a...)
}

func Debug(a ...any) {
	log.Debug(a...)
}

func Debugf(format string, a ...any) {
	log.Debugf(format, a...)
}

func Info(a ...any) {
	log.Info(a...)
}

func Infof(format string, a ...any) {
	log.Infof(format, a...)
}

func Warn(a ...any) {
	log.Warn(a...)
}

func Warnf(format string, a ...any) {
	log.Warnf(format, a...)
}

func Error(a ...any) {
	log.Error(a...)
}

func Errorf(format string, a ...any) {
	log.Errorf(format, a...)
}

func Fatal(a ...any) {
	log.Fatal(a...)
}

func Fatalf(format string, a ...any) {
	log.Fatalf(format, a...)
}

func Panic(a ...any) {
	log.Panic(a...)
}

func Panicf(format string, a ...any) {
	log.Panicf(format, a...)
}
