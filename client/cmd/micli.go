package main

import (
	"encoding/json"
	"fmt"

	//"net/http"
	"os"
	"strings"

	"ninego/log"
	"xiaobot/miservice"
)

func usage() {
	fmt.Printf("MiService - XiaoMi Cloud Service\n")
	fmt.Printf("Usage: The following variables must be set:\n")
	fmt.Printf("           export MI_USER=<Username>\n")
	fmt.Printf("           export MI_PASS=<Password>\n")
	fmt.Printf("           export MI_DID=<Device ID|Name>\n\n")
	fmt.Printf(miservice.IOCommandHelp("", os.Args[0]+" "))
}

func main() {
	args := os.Args
	argCount := len(args)

	//verboseFlag := false
	//verboseIndex := 4
	argIndex := 1

	if argCount > argIndex {
		//client := &http.Client{}
		account := miservice.NewAccount(
			//client,
			os.Getenv("MI_USER"),
			os.Getenv("MI_PASS"),
			//miservice.NewTokenStore(fmt.Sprintf("%s/.mi.token", os.Getenv("HOME"))),
			miservice.NewTokenStore("./.mi.token"),
		)

		var result interface{}
		var err error
		cmd := strings.Join(args[argIndex:], " ")

		if strings.HasPrefix(cmd, "mina") {
			service := miservice.NewAIService(account)
			deviceList, err := service.DeviceList(0)
			if err == nil && len(cmd) > 4 {
				_, err = service.SendMessage(deviceList, -1, cmd[4:], nil)
				result = "Message sent"
			} else {
				result = deviceList
			}
		} else {
			service := miservice.NewIOService(account, nil)
			result, err = miservice.IOCommand(service, os.Getenv("MI_DID"), cmd, os.Args[0]+" ")
		}

		if err != nil {
			log.Error("", err)
		} else {
			if resStr, ok := result.(string); ok {
				log.Println("string:", resStr)
			} else {
				log.Printf("%T=%#v\n", result, result)
				resBytes, _ := json.MarshalIndent(result, "", "  ")
				log.Println("json:", string(resBytes))
			}
		}
	} else {
		usage()
	}
}
