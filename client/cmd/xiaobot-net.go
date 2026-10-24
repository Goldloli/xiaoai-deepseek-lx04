package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ninego/log"
	"ninego/log/filelogger"
	"xiaobot"
	"xiaobot/monitor"
	"xiaobot/webui"
)

func display_banner() {
	fmt.Println("                                                                   ")
	fmt.Println("             o8o                       .o8                     .   ")
	fmt.Println("             `\"'                      \"888                   .o8   ")
	fmt.Println("oooo    ooo oooo   .oooo.    .ooooo.   888oooo.   .ooooo.  .o888oo ")
	fmt.Println(" `88b..8P'  `888  `P  )88b  d88' `88b  d88' `88b d88' `88b   888   ")
	fmt.Println("   Y888'     888   .oP\"888  888   888  888   888 888   888   888   ")
	fmt.Println(" .o8\"'88b    888  d8(  888  888   888  888   888 888   888   888 . ")
	fmt.Println("o88'   888o o888o `Y888\"\"8o `Y8bod8P'  `Y8bod8P' `Y8bod8P'   \"888\" ")
	fmt.Println("                                                                   ")
	fmt.Println("                                                                   ")
	fmt.Println("                                                                   ")
}

func main() {
	var sigch chan os.Signal

	defer func() {
		if r := recover(); r != nil {
			log.Error("panic:", r)
			if sigch != nil {
				close(sigch)
			}
			log.Close()
			os.Exit(1)
		}
	}()

	display_banner()

	cfgPath := flag.String("c", "config.json", "Config file path")
	debug := flag.String("d", "off", "filelogger Debug model: trace/info/warn/error/off")
	iface := flag.String("n", "", "monitor Net interface: auto or br-lan,eth0...")
	wAddr := flag.String("w", webui.WebAddr, "Webaddr:port (\"-\" is disable)")
	webOpen := flag.Bool("webui", false, "open webui config server")
	flag.Parse()

	if *debug != "off" {
		//log.SetLogger(zap.NewZapSugaredLogger())
		log.SetLogger(filelogger.NewSplitFilesLogger())
		switch *debug {
		case "trace":
			log.SetLevel(log.LevelDebug)
		case "info":
			log.SetLevel(log.LevelInfo)
		case "warn":
			log.SetLevel(log.LevelWarn)
		case "error":
			log.SetLevel(log.LevelError)
		}
	}

	config, err := xiaobot.NewConfigFromFile(*cfgPath)
	if err != nil || (config.NeedSetup()) || *webOpen {
		if err != nil {
			log.Println("加载配置文件失败：", err)
		}
		log.Println("进入配置服务,请打开Web页面去配置xiaobot参数...")
		webui.StartWebuiServer(config, *wAddr, true)
	} else if *wAddr != "-" {
		log.Println("启动配置服务,可打开Web页面去配置xiaobot参数...")
		webui.StartWebuiServer(config, *wAddr, false)
	}
	if config.Verbose {
		log.SetLevel(log.LevelDebug)
	}

	bot := xiaobot.NewMiBot(config)
	if *wAddr != "-" {
		//更新config
		go func() {
			for {
				select {
				case <-webui.WebDone:
					if config.Verbose {
						log.SetLevel(log.LevelDebug)
					}
					if err := bot.InitAllData(); err != nil {
						panic(err)
					}
					log.Println("update config")
				}
			}
		}()
		//平滑退出服务
		go func() {
			sigch = make(chan os.Signal)
			signal.Notify(sigch, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
			<-sigch
			webui.ShutdownWebuiServer()
			log.Close()
			os.Exit(0)
		}()
	}
	if err := bot.InitAllData(); err != nil {
		panic(err)
	}
	if *iface != "" {
		err = monitor.Run(bot, "", *iface)
		if err != nil {
			panic(err)
		}
	} else {
		err = bot.Run(0xFFFF)
		if err != nil {
			panic(err)
		}
	}
}
