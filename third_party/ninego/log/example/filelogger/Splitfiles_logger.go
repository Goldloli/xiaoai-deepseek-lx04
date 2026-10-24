package filelogger

import (
	"os"
	"path/filepath"

	"ninego/filelog"
	. "ninego/log"
)

var _ LoggerInterface = (*SplitFilesLogger)(nil)

// SplitFilesLogger 是日志接口的控制台实现
type SplitFilesLogger struct {
	logger *filelog.FileLogger
}

// NewSplitFilesLogger 创建一个新的控制台日志实例
func NewSplitFilesLogger() *SplitFilesLogger {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	filelogger := SplitFilesLogger{
		logger: filelog.NewDefaultLogger(dir+"/log", "log", ""),
	}
	filelogger.logger.SetLogLevel(filelog.INFO)
	filelogger.logger.SetLogCaller(true)
	filelogger.logger.SetAddSkipCaller(3)
	filelogger.logger.SetLogConsole(true)
	return &filelogger
}

func levelToFileLogLevel(level Level) filelog.LEVEL {
	levelMap := map[Level]filelog.LEVEL{
		LevelDebug: filelog.DEBUG,
		LevelInfo:  filelog.INFO,
		LevelWarn:  filelog.WARN,
		LevelError: filelog.ERROR,
		LevelFatal: filelog.FATAL,
	}
	if filelogLevel, ok := levelMap[level]; ok {
		return filelogLevel
	}
	return filelog.OFF
}

// SetLevel 设置日志级别
func (c *SplitFilesLogger) SetLevel(level Level) {
	c.logger.SetLogLevel(levelToFileLogLevel(level))
}

// GetLevel 获取当前日志级别
/*func (c *SplitFilesLogger) GetLevel() Level {
	return c.logger.GetLevel()
}*/

// Debug 输出调试级别日志
func (c *SplitFilesLogger) Debug(message string, v ...interface{}) {
	c.Log(filelog.DEBUG, message, v...)
}

// Info 输出信息级别日志
func (c *SplitFilesLogger) Info(message string, v ...interface{}) {
	c.Log(filelog.INFO, message, v...)
}

// Warn 输出警告级别日志
func (c *SplitFilesLogger) Warn(message string, v ...interface{}) {
	c.Log(filelog.WARN, message, v...)
}

// Error 输出错误级别日志
func (c *SplitFilesLogger) Error(message string, v ...interface{}) {
	c.Log(filelog.ERROR, message, v...)
}

// Panic 输出错误级别日志
func (c *SplitFilesLogger) Panic(message string, v ...interface{}) {
	c.Log(filelog.PANIC, message, v...)
}

// Fatal 输出致命级别日志并退出
func (c *SplitFilesLogger) Fatal(message string, v ...interface{}) {
	c.Log(filelog.FATAL, message, v...)
}

// Log 输出指定级别的日志
func (c *SplitFilesLogger) Log(level filelog.LEVEL, message string, fields ...interface{}) {
	if level < c.logger.GetLevel() {
		return
	}

	v := ArgsToKeyValues(fields...)
	// 根据级别选择输出流
	switch level {
	case filelog.DEBUG:
		c.logger.Debug(message, v...)
	case filelog.INFO:
		c.logger.Info(message, v...)
	case filelog.WARN:
		c.logger.Warn(message, v...)
	case filelog.ERROR:
		c.logger.Error(message, v...)
	case filelog.PANIC:
		c.logger.Panic(message, v...)
	case filelog.FATAL:
		c.logger.Fatal(message, v...)
	default:

	}
}

// Close 关闭日志，控制台日志无需释放资源
func (c *SplitFilesLogger) Close() error {
	return c.logger.Close()
}
