package filelog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// Receive logStr from f's logChan and print logstr to file
func (f *FileLogger) logWriter() {
	defer func() {
		if err := recover(); err != nil {
			if f.lg != nil { //close通道时lg也被置为了nil
				log.Printf("FileLogger's LogWritter() catch panic: %v\n", err)
			}
		}
	}()

	for {
		select {
		case str := <-f.logChan:

			f.p(str)
		}
	}
}

// print log
func (f *FileLogger) p(str string) {
	f.fileCheck()

	f.mu.RLock()
	defer f.mu.RUnlock()

	f.lg.Output(2, str)
	f.pc(str)
}

// print log in console, default log string wont be print in console
// NOTICE: when console is on, the process will really slowly
func (f *FileLogger) pc(str string) {
	if f.logConsole {
		if log.Prefix() != f.prefix {
			log.SetPrefix(f.prefix)
		}
		log.Println(str)
	}
}

// Printf throw logstr to channel to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (f *FileLogger) Printf(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1 + f.skipCaller) //calldepth=2
	f.logChan <- fmt.Sprintf("[%v:%v] ", filepath.Base(file), line) + fmt.Sprintf(format, v...)
}

// Print throw logstr to channel to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (f *FileLogger) Print(v ...interface{}) {
	_, file, line, _ := runtime.Caller(1 + f.skipCaller) //calldepth=2
	f.logChan <- fmt.Sprintf("[%v:%v] ", filepath.Base(file), line) + fmt.Sprint(v...)
}

// Println throw logstr to channel to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (f *FileLogger) Println(v ...interface{}) {
	_, file, line, _ := runtime.Caller(1 + f.skipCaller) //calldepth=2
	f.logChan <- fmt.Sprintf("[%v:%v] ", filepath.Base(file), line) + fmt.Sprintln(v...)
}

// ======================================================================================================================
// Debug log
func (f *FileLogger) Debugf(format string, v ...interface{}) {
	f.Log(DEBUG, true, format, v...)
}

// same with Debug()
func (f *FileLogger) Debug(message string, v ...interface{}) {
	f.Log(DEBUG, false, message, v...)
}

// Trace log
func (f *FileLogger) Tracef(format string, v ...interface{}) {
	f.Log(TRACE, true, format, v...)
}

// same with Trace()
func (f *FileLogger) Trace(message string, v ...interface{}) {
	f.Log(TRACE, false, message, v...)
}

// info log
func (f *FileLogger) Infof(format string, v ...interface{}) {
	f.Log(INFO, true, format, v...)
}

// same with Info()
func (f *FileLogger) Info(message string, v ...interface{}) {
	f.Log(INFO, false, message, v...)
}

// warning log
func (f *FileLogger) Warnf(format string, v ...interface{}) {
	f.Log(WARN, true, format, v...)
}

// same with Warn()
func (f *FileLogger) Warn(message string, v ...interface{}) {
	f.Log(WARN, false, message, v...)
}

// error log
func (f *FileLogger) Errorf(format string, v ...interface{}) {
	f.Log(ERROR, true, format, v...)
}

// same with Error()
func (f *FileLogger) Error(message string, v ...interface{}) {
	f.Log(ERROR, false, message, v...)
}

// Panic log
func (f *FileLogger) Panicf(format string, v ...interface{}) {
	f.Log(PANIC, true, format, v...)
	panic(fmt.Sprintf(format, v...))
}

// same with Panic()
func (f *FileLogger) Panic(message string, v ...interface{}) {
	f.Log(PANIC, false, message, v...)
	panic(message)
}

// Fatal log
func (f *FileLogger) Fatalf(format string, v ...interface{}) {
	defer f.Close()
	f.Log(FATAL, true, format, v...)
	os.Exit(1)
}

// same with Fatal()
func (f *FileLogger) Fatal(message string, v ...interface{}) {
	defer f.Close()
	f.Log(FATAL, false, message, v...)
	os.Exit(1)
}

// Log 输出指定级别的日志
func (f *FileLogger) Log(level LEVEL, format bool, message string, args ...interface{}) {
	//fmt.Sprint(args...)
	//fmt.Println(args)
	//return
	if level < f.logLevel {
		return
	}
	var logLine string
	switch level {
	case DEBUG:
		logLine = "[DEBUG] "
	case TRACE:
		logLine = "[TRACE] "
	case INFO:
		logLine = "[INFO] "
	case WARN:
		logLine = "[WARN] "
	case ERROR:
		logLine = "[ERROR] "
	case FATAL:
		logLine = "[FATAL] "
	default:
		return
	}
	if f.logCaller {
		_, file, line, _ := runtime.Caller(2 + f.skipCaller) //calldepth
		logLine += fmt.Sprintf("[%v:%v] ", filepath.Base(file), line)
	}
	if format {
		logLine += fmt.Sprintf(message, args...)
	} else {
		logLine += message
		/*for _, v := range args {
			logLine += fmt.Sprintf(" %#v", v)
		}*/
		n := len(args) - 1
		i := 0
		for {
			if i > n {
				break
			}
			field := args[i]
			if i <= n-1 {
				logLine += fmt.Sprintf(" %v=%+v", fmt.Sprint(field), args[i+1])
				i += 2
			} else {
				logLine += fmt.Sprintf(" %T=%+v", field, field)
				i += 1
			}
		}

	}
	//fmt.Println(logLine)
	f.logChan <- fmt.Sprint(logLine)
}
