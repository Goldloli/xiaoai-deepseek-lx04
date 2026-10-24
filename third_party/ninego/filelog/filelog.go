package filelog

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"path/filepath"
)

var (
	DATEFORMAT string = "2006-01-02"
)

var (
	DEFAULT_FILE_COUNT  int   = 0     //默认最大分割文件个数(0=不限)
	DEFAULT_FILE_SIZE   int64 = 0     //默认文件分割尺寸（0=不分割）
	DEFAULT_FILE_UNIT         = MB    //默认分割文件尺寸单位
	DEFAULT_LOG_SCAN    int64 = 300   //默认文件检查周期（5分钟）
	DEFAULT_LOG_SEQ     int   = 5000  //默认缓存通道数
	DEFAULT_LOG_LEVEL         = OFF   //TRACE //默认日志级别
	DEFAULT_LOG_CONSOLE       = false //默认是否向控制台输出
	DEFAULT_LOG_CALLER        = false //默认是否记录调用代码
)

type UNIT int64

const (
	_       = iota
	KB UNIT = 1 << (iota * 10)
	MB
	GB
	TB
)

type LEVEL byte

const (
	DEBUG LEVEL = iota
	TRACE
	INFO
	WARN
	ERROR
	PANIC
	FATAL
	OFF
)

type FileLogger struct {
	mu        *sync.RWMutex
	fileDir   string //日志目录
	fileName  string //日志文件名
	prefix    string //文件前缀
	suffix    int    //分割文件后缀起始值偏移
	fileCount int    //最大分割文件个数(0=不限)
	fileSize  int64  //文件分割尺寸（0=不分割）

	date *time.Time

	logFile *os.File
	lg      *log.Logger

	logScan int64 //文件检查周期（秒）

	logChan chan string //缓存通道

	logLevel   LEVEL //日志级别
	logConsole bool  //是否控制台显示

	logCaller  bool //是否显示调用代码来源
	skipCaller int  //调用深度调整
}

// NewDefaultLogger return a logger split by fileSize by default
func NewDefaultLogger(fileDir, fileName, prefix string) *FileLogger {
	defaultLogger := &FileLogger{
		mu:         new(sync.RWMutex),
		fileDir:    fileDir,
		fileName:   fileName,
		fileCount:  DEFAULT_FILE_COUNT,
		fileSize:   DEFAULT_FILE_SIZE * int64(DEFAULT_FILE_UNIT),
		prefix:     prefix,
		suffix:     0,
		logScan:    DEFAULT_LOG_SCAN,
		logChan:    make(chan string, DEFAULT_LOG_SEQ),
		logLevel:   DEFAULT_LOG_LEVEL,
		logConsole: DEFAULT_LOG_CONSOLE,
		logCaller:  DEFAULT_LOG_CALLER,
		skipCaller: 0,
	}

	defaultLogger.initLogger()

	return defaultLogger
}

func (f *FileLogger) GetLevel() LEVEL {
	return f.logLevel
}

func (f *FileLogger) initLogger() {
	t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
	f.date = &t

	f.mu.Lock()
	defer f.mu.Unlock()

	if !IsExist(f.fileDir) {
		os.Mkdir(f.fileDir, 0777 /*0755*/)
	}

	if f.logLevel < OFF {
		/*logFile := filepath.Join(f.fileDir, f.fileName+f.date.Format(DATEFORMAT)+".log")
		if !f.isMustSplit() {
			f.logFile, _ = os.OpenFile(logFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
			f.lg = log.New(f.logFile, f.prefix, log.LstdFlags|log.Lmicroseconds)
		} else {
			f.split()
		}*/
		f.fileCheck()
	}

	go f.logWriter()
	go f.fileMonitor()
}

// used for determine the fileLogger f is time to split.
// size: once the current fileLogger's fileSize >= config.fileSize need to split
// daily: once the current fileLogger stands for yesterday need to split
func (f *FileLogger) isMustSplit() bool {
	return f.isMustSplitByDate() || f.isMustSplitBySize() || f.logFile == nil
}
func (f *FileLogger) isMustSplitByDate() bool {
	if !(time.Now().Format(DATEFORMAT) == (*f.date).Format(DATEFORMAT)) {
		return true
	}
	return false
}
func (f *FileLogger) isMustSplitBySize() bool {
	if f.fileSize > 0 {
		logFile := filepath.Join(f.fileDir, f.fileName+f.date.Format(DATEFORMAT)+".log")
		if f.fileCount > 1 {
			if FileSize(logFile) >= f.fileSize {
				return true
			}
		}
	}
	return false
}

// Split fileLogger
func (f *FileLogger) split() {

	logFile := filepath.Join(f.fileDir, f.fileName+time.Now().Format(DATEFORMAT)+".log")

	if f.isMustSplitByDate() {
		if f.logFile != nil {
			f.logFile.Close()
		}

		t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
		f.date = &t

		//f.logFile, _ = os.Create(logFile)
		f.logFile, _ = os.OpenFile(logFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		f.lg = log.New(f.logFile, f.prefix, log.LstdFlags|log.Lmicroseconds)
	}

	if f.isMustSplitBySize() {
		f.suffix = int(f.suffix%f.fileCount + 1)
		if f.logFile != nil {
			f.logFile.Close()
		}

		logFileBak := logFile + "." + strconv.Itoa(f.suffix)
		if IsExist(logFileBak) {
			os.Remove(logFileBak)
		}
		os.Rename(logFile, logFileBak)

		if IsExist(logFile) {
			f.logFile, _ = os.OpenFile(logFile, os.O_RDWR|os.O_APPEND /*|os.O_CREATE*/, 0666)
		} else {
			f.logFile, _ = os.Create(logFile)
		}
		f.lg = log.New(f.logFile, f.prefix, log.LstdFlags|log.Lmicroseconds)
	}

	if f.logFile == nil {
		f.logFile, _ = os.OpenFile(logFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		f.lg = log.New(f.logFile, f.prefix, log.LstdFlags|log.Lmicroseconds)
	}

}

// After some interval time, goto check the current fileLogger's size or date
func (f *FileLogger) fileMonitor() {
	defer func() {
		if err := recover(); err != nil {
			f.lg.Printf("FileLogger's FileMonitor() catch panic: %v\n", err)
		}
	}()

	timer := time.NewTicker(time.Duration(f.logScan) * time.Second)
	for {
		select {
		case <-timer.C:
			if f.logLevel < OFF {
				f.fileCheck()
			}
		}
	}
}

// If the current fileLogger need to split, just split
func (f *FileLogger) fileCheck() {
	defer func() {
		if err := recover(); err != nil {
			f.lg.Printf("FileLogger's FileCheck() catch panic: %v\n", err)
		}
	}()

	if f.isMustSplit() {
		f.mu.Lock()
		defer f.mu.Unlock()

		f.split()
	}
}

// passive to close fileLogger
func (f *FileLogger) Close() error {
	time.Sleep(time.Second)

	close(f.logChan)
	f.lg = nil

	return f.logFile.Close()
}

// 判断文件或文件夹是否存在
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// return length in bytes for regular files
func FileSize(file string) int64 {
	f, e := os.Stat(file)
	if e != nil {
		return 0
	}
	return f.Size()
}
