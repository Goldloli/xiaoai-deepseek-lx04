package filelog

import (
	"testing"
	//"time"
)

func TestFilelog(t *testing.T) {
	lg := NewDefaultLogger("", "log", "")
	lg.SetLogConsole(true)
	lg.SetLogCaller(true)
	lg.Info("this is a test.")
	lg.Infof("this is a %s test%v.", "hello", 2)
	lg.Info("this is a test.", lg)
	lg.Info("this is a test.", "aaa", 1, "bbb", "2")
	lg.Info("this is a test.")
	lg.Debug("debffug", "test")
	lg.Trace("--end--")
	lg.Print("test","this",555)
	//time.Sleep(time.Second)
	lg.Close()

}
