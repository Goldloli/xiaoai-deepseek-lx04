// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ninego/log"
	"xiaobot"
	"xiaobot/jsengine"
	"xiaobot/log/filelogger"
	"xiaobot/music"
	"xiaobot/runtimeenv"
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
	fmt.Println("                              ~anxu~                               ")
	fmt.Println("                                                                   ")
}

func main() {
	if err := runtimeenv.ConfigureDNSFromEnv(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	var sigch chan os.Signal

	defer func() {
		if err := jsengine.SaveStorageToFile(); err != nil {
			log.Println("save botdata error:", err)
		}
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

	// 设置东八区时区
	xiaobot.FixedZone()

	cfgPath := flag.String("c", "config.json", "加载指定的config配置文件")
	debug := flag.String("d", "off", "日志文件记录 model: off(关闭)/info/warn/error/trace")
	trigger := flag.Bool("t", false, "启用监控触发模式（需<mac_monitor.sh>脚本配合）\n注意: 未指定-t参数时默认启用的是轮询模式")
	wAddr := flag.String("w", ":"+webui.WebPort, "指定web服务的端口号,格式为\"[IP:]端口\" (或用\"-\"横线关闭web服务)")
	cfgOpen := flag.Bool("config", false, "启动后进入config配置页面")
	webOpen := flag.Bool("webui", false, "启动后进入chat对话页面")
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

	log.Println("加载配置文件:", *cfgPath)
	config, err := xiaobot.NewConfigFromFile(*cfgPath)
	if err != nil || (config.NeedSetup()) || *cfgOpen || *webOpen {
		if err != nil {
			log.Println("加载配置文件失败：", err)
		}
		if config.NeedSetup() || *cfgOpen {
			log.Println("进入配置服务,请打开Web页面去配置xiaobot参数...")
			webui.StartWebuiServer(config, *wAddr, 1)
		} else {
			webui.StartWebuiServer(config, *wAddr, 2)
		}
	} else if *wAddr != "-" {
		log.Println("启动配置服务,可打开Web页面去配置xiaobot参数...")
		webui.StartWebuiServer(config, *wAddr, 0)
	}
	if config.Verbose {
		log.SetLevel(log.LevelDebug)
	}

	if err := jsengine.LoadStorageFromFile(); err != nil {
		log.Println("load botdat error:", err)
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
			//return
			if err := jsengine.SaveStorageToFile(); err != nil {
				log.Println("save botdata error:", err)
			}
			log.Close()
			os.Exit(0)
		}()
	}
	if err := bot.InitAllData(); err != nil {
		panic(err)
	}
	if webui.WebAddr == "" {
		srvip, err := webui.CheckServerIP(bot.Box.IP_Addr)
		if err != nil {
			log.Error("Music Server:", err)
		}
		webui.WebAddr = srvip
	}
	if webui.WebAddr == "" {
		log.Println("音乐http服务未能开启,请添加-w启动参数指定本机IP地址。")
	} else {
		music.PlayerSvrIP = webui.WebAddr + ":" + webui.WebPort
		log.Printf("http://%s\n", music.PlayerSvrIP)
		log.Printf("ws://%s/ws\n", music.PlayerSvrIP)
	}

	//加载计划任务
	jsengine.Schedules.FetchFromExecDir()
	jsengine.Schedules.Start()
	defer jsengine.Schedules.Stop()

	if *trigger {
		webui.RunChat(bot)
		if err = webui.Run(0); err != nil {
			panic(err)
		}
	} else {
		webui.RunChat(bot)
		if err := bot.Run(0xFFFF); err != nil {
			panic(err)
		}
	}
}
