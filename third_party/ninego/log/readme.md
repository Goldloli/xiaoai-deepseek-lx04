### 支持自定义的日志接口
项目实现可自行替换任意日志器的日志接口，主要解决不同项目使用不同log输出不一致的问题。

提供`Print/Printf/Println`等类似标准库的函数，方便调试使用，同时支持全局日志对象和独立日志实例。支持日志字段（Fields），满足基础日志需求。

实现了常见的日志级别（Debug/Info/Warn/Error/Panic/Fatal），通过`LoggerInterface`定义了统一的日志接口规范，支持自定义实现（如控制台日志、文件日志等），具备良好的扩展性。

默认提供了ConsoleLogger控制台日志器，具有抽象日志格式化接口（`Formatter`），输出格式灵活，支持自定义格式（如JSON、文本）。
可根据需要进一步实现`FileLogger`、`Zap`、`logrus`等具体日志器，只需实现`LoggerInterface`接口即可替换控制台日志接入系统。


logger 接口须实现以下方法：

``` golang
// Logger 定义日志接口
type LoggerInterface interface {
	// 设置日志级别
	SetLevel(level Level)

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

```
### log 基础使用
#### logger的基本方法
```golang
//记录日志
func Debug(msg string, v ...interface{})
func Info(msg string, v ...interface{}) 
func Warn(msg string, v ...interface{}) 
func Error(msg string, v ...interface{})
func Panic(msg string, v ...interface{})
func Fatal(msg string, v ...interface{})

//控制台输出，同时记录为Info日志（当作log使用）
func Errorf(format string, v ...interface{})
func Printf(format string, v ...interface{})
func Println(v ...interface{})
//控制台输出，同时记录为Info日志（当作fmt使用）
func Print(v ...interface{})
```
log支持采用...Fields 做为日志的传入参数（map[string]interface{}）
```
// Fields 定义日志字段类型
type Fields map[string]interface{}
```
使用：
```
logger.Debug("这是一个调试日志", logger.Fields{"step": "初始化"})
logger.Info( "用户登录", logger.Fields{"user_id": "123", "ip": "192.168.1.1"})
logger.Warn( "磁盘空间不足", logger.Fields{"used": "85%", "threshold": "90%"})
logger.Error("数据库连接失败", logger.Fields{"error": "connection timeout", "retry_count": 3})
```

提供了1个ArgsToKeyValues函数，可以在将Fields参数转换为Key-Value形式：
```
// 将Args转换为Key-Value结对的...interface{} 转换后格式为[key1, value1, key2, value2, ...]
func ArgsToKeyValues(args ...interface{}) (kv []interface{}) {
```

##### log 切换
通过SetLogger方法切换Logger,该logger须实现LoggerInterface接口的所有方法(SetLevel/Debug/Info/Warn/Error/Fatal/Close)。
```golang
func SetLogger(in LoggerInterface)
```
注意,SetLogger切换后，该logger的日志等级会同步更新为当前log设置的Level，而不是原logger的日志等级。

##### log 切换日志等级

通过SetLevel方法切换
``` golang
func SetLevel(level Level)
```
level定义包括

```golang
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
```

全局变量globalog是当前使用的日志对象，它默认创建使用的是ConsoleLogger控制台日志，可通过SetLogger设置成其他的logger。
``` golang
// 默认的日志级别为info
var globalog = newlogger(LevelInfo)
```

#### ConsoleLogger
控制台日志

引用logger后默认开启的日志，在控制台显示日志内容。

``` golang
// 全局日期格式定义
var TimeFormat = "2006-01-02 15:04:05.000"

// 抽象日志格式化接口（`Formatter`），支持自定义控制台输出格式
type Formatter interface {
	Format(level Level, timestamp time.Time, message string, file string, line int, fields ...interface{}) string
}
```
可以通过 SetConsoleFormatter 设置自已的控制台显示格式。


### 使用演示
```
package main

import (
	"gitee.com/ninego/log/logger"
)

func main() {
	//logger.SetLogger(其他的logger)
	defer func() {
		logger.Close() 
    }()
    
	logger.Println("Hello World", "在控制台显示", "同时也记录到Info日志", "当做log使用")
	logger.Printf("在控制台显示，同时记录到`%s`日志\n", "Info")
	logger.Print("在控制台显示，同时记录到Info日志", "当做fmt使用\n")
	// 演示不同级别的日志
	logger.Debug("这是一个调试日志", logger.Fields{"step": "初始化"})
	logger.Info( "用户登录", logger.Fields{"user_id": "123", "ip": "192.168.1.1"})
	logger.Warn( "磁盘空间不足", logger.Fields{"used": "85%", "threshold": "90%"})
	logger.Error("数据库连接失败", logger.Fields{"error": "connection timeout", "retry_count": 3})
}
```
### 不同的logger文件例子
```
$ cd example
$ go run main.go
```
包含FileLogger、ZapSugaredLogger 2个log日志器的使用实现，以及默认的控制台打印日志示例。