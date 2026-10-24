package webui

import (
	"context"
	"fmt"
	"net/http"
	"ninego/log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
	"xiaobot"
)

var (
	config *xiaobot.Config
	bot    *xiaobot.MiBot

	server *http.Server
	ticker *time.Ticker

	WebAddr string = ""
	WebPort string = "9997"

	WebDone chan bool
)

const (
	monitoringTriggerInterval = 5 * time.Second //每次触发给5秒延时轮询时间
)

func StartWebuiServer(cfg *xiaobot.Config, wAddr string, show int) {
	config = cfg

	if strings.IndexByte(wAddr, ':') == -1 {
		_, err := strconv.Atoi(wAddr)
		if err == nil {
			WebPort = wAddr
			WebAddr = ""
		} else {
			WebPort = ""
			WebAddr = wAddr
		}
	} else {
		WebAddr = strings.Split(wAddr, ":")[0] // 地址部分
		WebPort = strings.Split(wAddr, ":")[1] // 端口部分
	}
	if WebPort == "" {
		WebPort = "9997"
	}

	if WebAddr != "" && WebAddr != "0.0.0.0" {
		wAddr = WebAddr + ":" + WebPort
	} else {
		wAddr = "127.0.0.1" + wAddr
	}
	//log.Printf("http://%s\n", wAddr)
	if show != 0 {
		go func() {
			time.Sleep(time.Second)
			if show == 1 {
				_ = open(fmt.Sprintf("http://%s/config", wAddr))
			} else {
				_ = open(fmt.Sprintf("http://%s/", wAddr))
			}
		}()
	}

	mux := Router()
	server = &http.Server{
		Addr:           WebAddr + ":" + WebPort,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   5 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println(err)
		}
	}()
	WebDone = make(chan bool) // 用于通知主程序退出
	if show == 1 {
		<-WebDone // 等待关闭完成通知
		log.Println("config done. ")
		/*cxt, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := server.Shutdown(cxt)
		if err != nil {
			fmt.Println("err", err)
		}*/
	} else {
		/*go func() {
			for {
				select {
				case <-WebDone:
					log.Println("update config")
				}
			}
		}()*/
	}
}

func ShutdownWebuiServer() {
	cxt, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := server.Shutdown(cxt)
	if err != nil {
		fmt.Println("err", err)
	}
	log.Println("shutdown web server.")
}

// open opens the specified URL in the default browser of the user.
func open(url string) error {
	var (
		cmd  string
		args []string
	)

	switch runtime.GOOS {
	case "windows":
		cmd, args = "cmd", []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default:
		// "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func RunChat(mt *xiaobot.MiBot) error {
	bot = mt
	return nil
}
func Run(monitor int) error {
	ticker = time.NewTicker(monitoringTriggerInterval)
	defer ticker.Stop()
	//每次触发给5秒延时轮询时间
	log.Println("音箱MAC地址:", bot.Box.MacAddr)
	go func() {
		for {
			<-ticker.C
			bot.UpdateMonitor(-1)
		}
	}()
	//运行小爱
	return bot.Run(monitor)
}
