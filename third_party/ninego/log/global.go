package log

import (
	"fmt"
	"reflect"
	"time"
)

// 全局日期格式定义
var TimeFormat = "2006-01-02 15:04:05.000"

// 全局 log 默认的日志级别Info,也可以单独 NewLogger 获取新的实例
var globalog = newlogger(LevelError)

func newlogger(level Level) *Logger {
	l := &Logger{logger: NewConsoleLogger(level), level: level}
	return l
}

func SetLogger(in LoggerInterface) {
	globalog.lock.Lock()
	defer globalog.lock.Unlock()
	globalog.logger = in
	globalog.logger.SetLevel(globalog.level)
}

func GetLogger() LoggerInterface {
	return globalog
}

// 将Args转换为...interface{} 转换后格式为[key1, value1, key2, value2, ...]
func ArgsToKeyValues(args ...interface{}) (kv []interface{}) {
	if args == nil || len(args) == 0 {
		return //nil
	}

	n := len(args) - 1
	i := 0
	for {
		if i > n {
			break
		}
		field := args[i]
		switch field.(type) {
		case Fields:
			for k, v := range field.(Fields) {
				kv = append(kv, k, v)
				i += 1
			}
		default:
			k := reflect.ValueOf(field).Kind()
			if k == reflect.Map || k == reflect.Slice || k == reflect.Array || k == reflect.Struct || k == reflect.Interface || k == reflect.Ptr {
				kv = append(kv, fmt.Sprintf("%T", field), field)
				i += 1
			} else {
				if i <= n-1 {
					kv = append(kv, fmt.Sprint(field), args[i+1])
					i += 2
				} else {
					kv = append(kv, fmt.Sprintf("%T", field), field)
					i += 1
				}
			}
		}
	}
	return
}

func Info(msg string, v ...interface{}) {
	globalog.logger.Info(msg, v...)
}

func Debug(msg string, v ...interface{}) {
	globalog.logger.Debug(msg, v...)
}

func Warn(msg string, v ...interface{}) {
	globalog.logger.Warn(msg, v...)
}
func Error(msg string, v ...interface{}) {
	globalog.logger.Error(msg, v...)
}

func Panic(msg string, v ...interface{}) {
	globalog.logger.Panic(msg, v...)
}

func Fatal(msg string, v ...interface{}) {
	globalog.logger.Fatal(msg, v...)
}

func SetLevel(level Level) {
	globalog.level = level
	globalog.logger.SetLevel(level)
}

func GetLevel(level Level) Level {
	return globalog.level
}

/*func With(v ...Interface{}) LoggerInterface {
	newLog := globalog.logger.With(fields...)
	l := &Logger{logger: newLog}
	return l
}*/

func Close() error {
	return globalog.logger.Close()
}

// Print calls Output to print to the standard logger.
// Arguments are handled in the manner of [fmt.Print].
func Print(v ...interface{}) {
	if len(v) > 1 {
		fmt.Print(v...)
		_, ok := (globalog.logger).(*ConsoleLogger)
		if !ok {
			globalog.logger.Info(fmt.Sprint(v[0]), v[1:]...)
		}
	} else {
		fmt.Print(fmt.Sprint(v[0]))
		_, ok := (globalog.logger).(*ConsoleLogger)
		if !ok {
			globalog.logger.Info(fmt.Sprint(v[0]))
		}
	}
}

// Printf calls Output to print to the standard logger.
// Arguments are handled in the manner of [fmt.Printf].
func Printf(format string, v ...interface{}) {
	fmt.Printf(time.Now().Format(TimeFormat)+" "+format, v...)
	_, ok := (globalog.logger).(*ConsoleLogger)
	if !ok {
		globalog.logger.Info(fmt.Sprintf(format, v...))
	}
}

// Println calls Output to print to the standard logger.
// Arguments are handled in the manner of [fmt.Println].
func Println(v ...interface{}) {
	if len(v) > 1 {
		tv := []any{time.Now().Format(TimeFormat)}
		tv = append(tv, v...)
		fmt.Println(tv...)
		_, ok := (globalog.logger).(*ConsoleLogger)
		if !ok {
			globalog.logger.Info(fmt.Sprint(v[0]), v[1:]...)
		}
	} else {
		fmt.Println(time.Now().Format(TimeFormat), fmt.Sprint(v[0]))
		_, ok := (globalog.logger).(*ConsoleLogger)
		if !ok {
			globalog.logger.Info(fmt.Sprint(v[0]))
		}
	}
}

func Infof(format string, v ...interface{}) {
	globalog.logger.Info(fmt.Sprintf(format, v...))
}

func Debugf(format string, v ...interface{}) {
	globalog.logger.Debug(fmt.Sprintf(format, v...))
}

func Warnf(format string, v ...interface{}) {
	globalog.logger.Warn(fmt.Sprintf(format, v...))
}

func Errorf(format string, v ...interface{}) {
	globalog.logger.Error(fmt.Sprintf(format, v...))
}

func Fatalf(format string, v ...interface{}) {
	globalog.logger.Fatal(fmt.Sprintf(format, v...))
}

// Panicf is equivalent to [Printf] followed by a call to panic().
func Panicf(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	globalog.logger.Panic(s)
	panic(s)
}

// Panicln is equivalent to [Println] followed by a call to panic().
func Panicln(v ...any) {
	s := fmt.Sprint(v[0])
	if len(v) > 1 {
		globalog.logger.Panic(s, v[1:]...)
	} else {
		globalog.logger.Panic(s)
	}
	panic(s)
}
