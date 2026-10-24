package log

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"time"
)

/*
这行代码是 Go 语言中一种常用的编译期接口实现检查技巧，用于验证 *ConsoleLogger 类型是否完整实现了 LoggerInterface 接口。
代码作用拆解：
  var _ LoggerInterface：声明一个类型为 LoggerInterface 的匿名变量（_ 表示忽略变量名，仅用于类型检查）。
  (*ConsoleLogger)(nil)：将 nil 转换为 *ConsoleLogger 类型的指针（此时它是一个类型为 *ConsoleLogger 的空指针）。
  尝试将这个指针赋值给 LoggerInterface 类型的变量。
核心目的：
  编译期验证：如果 *ConsoleLogger 没有完全实现 LoggerInterface 接口的所有方法，这行代码会在编译时触发错误，提醒开发者补全实现。
  无运行时开销：由于使用了 _ 且赋值为 nil，这段代码不会分配实际内存，仅用于编译阶段的类型检查。
*/
var _ LoggerInterface = (*ConsoleLogger)(nil)

// ConsoleLogger 是日志接口的控制台实现
type ConsoleLogger struct {
	level Level
}

// NewConsoleLogger 创建一个新的控制台日志实例
func NewConsoleLogger(level Level) *ConsoleLogger {
	return &ConsoleLogger{
		level: level,
	}
}

type Formatter interface {
	Format(level Level, timestamp time.Time, message string, file string, line int, fields ...interface{}) string
}

var formatter Formatter = nil // 输出格式Formatter

func SetConsoleFormatter(f Formatter) {
	formatter = f
}

// SetLevel 设置日志级别
func (c *ConsoleLogger) SetLevel(level Level) {
	c.level = level
}

// GetLevel 获取当前日志级别
func (c *ConsoleLogger) GetLevel() Level {
	return c.level
}

// Debug 输出调试级别日志
func (c *ConsoleLogger) Debug(message string, v ...interface{}) {
	c.Log(LevelDebug, message, v...)
}

// Info 输出信息级别日志
func (c *ConsoleLogger) Info(message string, v ...interface{}) {
	c.Log(LevelInfo, message, v...)
}

// Warn 输出警告级别日志
func (c *ConsoleLogger) Warn(message string, v ...interface{}) {
	c.Log(LevelWarn, message, v...)
}

// Error 输出错误级别日志
func (c *ConsoleLogger) Error(message string, v ...interface{}) {
	c.Log(LevelError, message, v...)
}

// Panic 输出致命级别日志并触发异常
func (c *ConsoleLogger) Panic(message string, v ...interface{}) {
	c.Log(LevelPanic, message, v...)
	Panic(message)
}

// Fatal 输出致命级别日志并退出
func (c *ConsoleLogger) Fatal(message string, v ...interface{}) {
	c.Log(LevelFatal, message, v...)
	os.Exit(1)
}

// Log 输出指定级别的日志
func (c *ConsoleLogger) Log(level Level, message string, fields ...interface{}) {
	if level < c.level {
		return
	}

	// 格式化时间
	//timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	//调用处代码
	_, file, line, _ := runtime.Caller(3) //calldepth=x+1

	var logLine string
	if formatter != nil {
		logLine = formatter.Format(level, time.Now(), message, filepath.Base(file), line, fields...)
	} else {
		// 格式化时间
		timestamp := time.Now().Format(TimeFormat)

		// 基础日志信息
		logLine = fmt.Sprintf("%s [%s] %s [%v:%v]", timestamp, level.String(), message, filepath.Base(file), line)

		// 添加日志字段
		if len(fields) > 0 {
			logLine += " | "
			n := len(fields) - 1
			i := 0
			for {
				if i > n {
					break
				}
				field := fields[i]
				switch field.(type) {
				case Fields:
					for k, v := range field.(Fields) {
						logLine += fmt.Sprintf("%s=%+v ", k, v)
						i += 1
					}
				default:
					k := reflect.ValueOf(field).Kind()
					//fmt.Println(k)
					if k == reflect.Map || k == reflect.Slice || k == reflect.Array || k == reflect.Struct || k == reflect.Interface || k == reflect.Ptr {
						logLine += fmt.Sprintf("%T=%+v ", field, field)
						i += 1
					} else {
						if i <= n-1 {
							logLine += fmt.Sprintf("%s=%+v ", field, fields[i+1])
							i += 2
						} else {
							logLine += fmt.Sprintf("%T=%+v ", field, field)
							i += 1
						}
					}
				}
			}
		}
	}

	// 根据级别选择输出流
	switch level {
	case LevelError, LevelFatal:
		fmt.Fprintln(os.Stderr, logLine)
	default:
		fmt.Println(logLine)
	}
}

// Close 关闭日志，控制台日志无需释放资源
func (c *ConsoleLogger) Close() error {
	return nil
}
