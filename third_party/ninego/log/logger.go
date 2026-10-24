package log

import (
	"sync"
)

// 单个日志对象
type Logger struct {
	lock   sync.Mutex
	level  Level
	logger LoggerInterface
}

// 也可以单独 NewLogger
func NewLogger(level Level) *Logger {
	return newlogger(level)
}

func (l *Logger) Debug(msg string, v ...interface{}) {
	l.logger.Debug(msg, v...)
}

func (l *Logger) Warn(msg string, v ...interface{}) {
	l.logger.Warn(msg, v...)
}

func (l *Logger) Error(msg string, v ...interface{}) {
	l.logger.Error(msg, v...)
}

func (l *Logger) Panic(msg string, v ...interface{}) {
	l.logger.Panic(msg, v...)
}

func (l *Logger) Fatal(msg string, v ...interface{}) {
	l.logger.Fatal(msg, v...)
}

func (l *Logger) Info(msg string, v ...interface{}) {
	l.logger.Info(msg, v...)
}

func (l *Logger) SetLevel(level Level) {
	l.level = level
	l.logger.SetLevel(level)
}

func (l *Logger) Close() error {
	return l.logger.Close()
}
