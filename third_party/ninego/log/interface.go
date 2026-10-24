package log

// Level 定义日志级别
type Level int

// 日志级别常量
const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelPanic
	LevelFatal
)

// String 返回日志级别的字符串表示
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelPanic:
		return "PANIC"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Fields 定义日志字段类型
type Fields map[string]interface{}

// Logger 定义日志接口
type LoggerInterface interface {
	// 设置日志级别
	SetLevel(level Level)

	// 获取当前日志级别
	//GetLevel() Level

	// 调试级别日志
	Debug(message string, v ...interface{})

	// 信息级别日志
	Info(message string, v ...interface{})

	// 警告级别日志
	Warn(message string, v ...interface{})

	// 错误级别日志
	Error(message string, v ...interface{})

	// 致命级别日志，输出后会触发异常
	Panic(message string, v ...interface{})

	// 致命级别日志，输出后会退出程序
	Fatal(message string, v ...interface{})

	// 关闭日志，释放资源
	Close() error
}

/*
	// 演示不同级别的日志
	a.log.Debug("这是一个调试日志", logger.Fields{"step": "初始化"})
	a.log.Info( "用户登录", logger.Fields{"user_id": "123", "ip": "192.168.1.1"})
	a.log.Warn( "磁盘空间不足", logger.Fields{"used": "85%", "threshold": "90%"})
	a.log.Error("数据库连接失败", logger.Fields{"error": "connection timeout", "retry_count": 3})
*/
