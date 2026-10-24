package main

import (
	"fmt"
	//"reflect"
	"time"

	logger "ninego/log"
	"ninego/log/example/filelogger"
	"ninego/log/example/zap"
)

type TxtFormat struct {
}

func (f *TxtFormat) Format(level logger.Level, timestamp time.Time, message string, file string, line int, fields ...interface{}) (logLine string) {
	//var logLine string

	// 格式化时间
	stimestamp := time.Now().Format(logger.TimeFormat) //("2006/01/02 15:04:05")

	// 基础日志信息
	logLine = fmt.Sprintf("%s [%s] %s [%v:%v]", stimestamp, level.String(), message, file, line)

	// 添加日志字段
	if len(fields) > 0 {
		logLine += " | "
		for _, v := range fields {
			logLine += fmt.Sprintf(" %+v ", v)
		}
	}

	return logLine + " :)"
}

func main() {
	defer func() {
		fmt.Println("end.")
		logger.Close()
	}()
	i := 0

	//切换到Debug日志
	//logger.SetLevel(logger.LevelError)
	logger.SetLevel(logger.LevelDebug)
	logger.SetConsoleFormatter(&TxtFormat{})

	logger.Print("在控制台显示", "当做fmt使用\n")
	logger.Printf("在控制台显示，同时记录到`%s`日志\n", "Info")
	logger.Println("Hello World", "在控制台显示", "同时也记录到Info日志")
	logger.Debug("这是一个调试日志", "Test")
	logger.Debug("这是一个调试日志", "test", "1", "test2")
	logger.Debug("这是一个调试日志", logger.Fields{"step": "初始化"})
	logger.Debug("这是一个调试日志", map[string]interface{}{"step": "初始化", "step2": "加载数据"}, logger.GetLogger(), "TEST", &i)
	logger.Info("用户登录", logger.Fields{"user_id": "123", "ip": "192.168.1.1"})
	logger.Warn("磁盘空间不足", logger.Fields{"used": "85%", "threshold": "90%"})
	logger.Error("数据库连接失败", logger.Fields{"error": "connection timeout", "retry_count": 3})
	//logger.Fatal("game over")

	//zap日志
	logger.SetLogger(zap.NewZapSugaredLogger())
	logger.Print("\nzap-----------\n")

	logger.Print("在控制台显示", "当做fmt使用\n")
	logger.Printf("在控制台显示，同时记录到`%s`日志\n", "Info")
	logger.Println("Hello World", "在控制台显示", "同时也记录到Info日志")
	logger.Debug("这是一个调试日志", "test", "1", "test2")
	logger.Debug("这是一个调试日志", logger.Fields{"step": "初始化"})
	logger.Debug("这是一个调试日志", map[string]interface{}{"step": "初始化", "step2": "加载数据"}, "TEST", logger.GetLogger(), &i)
	logger.Info("用户登录", logger.Fields{"user_id": "123", "ip": "192.168.1.1"})
	logger.Warn("磁盘空间不足", logger.Fields{"used": "85%", "threshold": "90%"})
	logger.Error("数据库连接失败", logger.Fields{"error": "connection timeout", "retry_count": 3})
	//logger.Fatal("game over")

	//filelogger日志
	logger.SetLogger(filelogger.NewSplitFilesLogger())
	logger.Print("\nfile-----------\n")

	logger.Print("在控制台显示", "当做fmt使用\n")
	logger.Printf("在控制台显示，同时记录到`%s`日志\n", "Info")
	logger.Println("Hello World", "在控制台显示", "同时也记录到Info日志")
	logger.Debug("这是一个调试日志", "test", "1", "test2")
	logger.Debug("这是一个调试日志", logger.Fields{"step": "初始化"})
	logger.Debug("这是一个调试日志", map[string]interface{}{"step": "初始化", "step2": "加载数据"}, "TEST", logger.GetLogger(), &i)
	logger.Info("用户登录", logger.Fields{"user_id": "123", "ip": "192.168.1.1"})
	logger.Warn("磁盘空间不足", logger.Fields{"used": "85%", "threshold": "90%"})
	logger.Error("数据库连接失败", logger.Fields{"error": "connection timeout", "retry_count": 3})
	//logger.Fatal("game over")
}
