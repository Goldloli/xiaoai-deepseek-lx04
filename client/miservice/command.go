package miservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var template = `Get Props: {prefix}<siid[-piid]>[,...]\n\
           {prefix}1,1-2,1-3,1-4,2-1,2-2,3\n\
Set Props: {prefix}<siid[-piid]=[#]value>[,...]\n\
           {prefix}2=#60,2-2=#false,3=test\n\
Do Action: {prefix}<siid[-piid]> <arg1|#NA> [...] \n\
           {prefix}2 #NA\n\
           {prefix}5 Hello\n\
           {prefix}5-4 Hello #1\n\n\
Call MIoT: {prefix}<cmd=prop/get|/prop/set|action> <params>\n\
           {prefix}action {quote}{{"did":"{did}","siid":5,"aiid":1,"in":["Hello"]}}{quote}\n\n\
Call MiIO: {prefix}/<uri> <data>\n\
           {prefix}/home/device_list {quote}{{"getVirtualModel":false,"getHuamiDevices":1}}{quote}\n\n\
Devs List: {prefix}list [name=full|name_keyword] [getVirtualModel=false|true] [getHuamiDevices=0|1]\n\
           {prefix}list Light true 0\n\n\
MIoT Spec: {prefix}spec [model_keyword|type_urn] [format=text|python|json]\n\
           {prefix}spec\n\
           {prefix}spec speaker\n\
           {prefix}spec xiaomi.wifispeaker.lx04\n\
           {prefix}spec urn:miot-spec-v2:device:speaker:0000A015:xiaomi-lx04:1\n\n\
MIoT Decode: {prefix}decode <ssecurity> <nonce> <data> [gzip]\n\
`

func IOCommandHelp(did string, prefix string) string {
	var quote string
	if prefix == "" {
		prefix = "?"
		quote = ""
	} else {
		quote = "'"
	}

	tmp := strings.ReplaceAll(template, "{prefix}", prefix)
	tmp = strings.ReplaceAll(tmp, "{quote}", quote)
	if did == "" {
		did = "267090026"
	}
	tmp = strings.ReplaceAll(tmp, "{did}", did)
	return tmp
}

func IOCommand(service *IOService, did string, text string, prefix string) (interface{}, error) {
	fmt.Println("DID=", did, "CMD=", text, "Args=", prefix)
	cmd, arg := twinsSplit(text, " ", "")
	if strings.HasPrefix(cmd, "/") {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(arg), &args); err != nil {
			fmt.Println("json error: ", arg)
			return nil, err
		}
		return service.Request(cmd, args)
	} //json中的引号cmd传入没了？ 解码失败。 这样可行(无外单引号,可用外双引号)：{\"getVirtualModel\":false,\"getHuamiDevices\":1}

	if strings.HasPrefix(cmd, "prop") || cmd == "action" {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(arg), &args); err != nil {
			return nil, err
		}
		return service.Request(cmd, args)
	}

	argv := []string{}
	if strings.TrimSpace(arg) != "" {
		argv = strings.Split(arg, " ")
	}
	//argv := strings.Split(strings.TrimSpace(arg), " ") Split空串得到长度为1的切片,其值为空串？？
	argc := len(argv)
	fmt.Println(arg, argc, argv)
	var arg0 string
	if argc > 0 {
		arg0 = argv[0]
	}
	var arg1 string
	if argc > 1 {
		arg1 = argv[1]
	}
	var arg2 string
	if argc > 2 {
		arg2 = argv[2]
	}
	switch cmd {
	case "list":
		a1 := false
		if arg1 != "" {
			a1, _ = strconv.ParseBool(arg1)
		}
		a2 := 0
		if arg2 != "" {
			a2, _ = strconv.Atoi(arg2)
		}
		return service.DeviceList(a1, a2) // Implement this method for the IOService
	case "spec":
		return service.IotSpec(arg0) // Implement this method for the IOService
	case "decode":
		if argc > 3 && argv[3] == "gzip" {
			return service.IotDecode(argv[0], argv[1], argv[2], true) // Implement this method for the IOService
		}
		return service.IotDecode(argv[0], argv[1], argv[2], false) // Implement this method for the IOService
	}

	if did == "" || cmd == "" || cmd == "help" || cmd == "-h" || cmd == "--help" {
		return IOCommandHelp(did, prefix), nil
	}

	if !isDigit(did) {
		devices, err := service.DeviceList(false, 0) // Implement this method for the IOService
		if err != nil {
			return nil, err
		}
		if len(devices) == 0 {
			return nil, errors.New("Device not found: " + did)
		}
		for _, device := range devices {
			if device.Name == did {
				did = device.Did
				break
			}
		}
	}

	var props [][]interface{}
	setp := true
	miot := 2 //null
	for _, item := range strings.Split(cmd, ",") {
		key, value := twinsSplit(item, "=", "")
		siid, iid := twinsSplit(key, "-", "1")
		var prop []interface{}
		if isDigit(siid) && isDigit(iid) {
			s, _ := strconv.Atoi(siid)
			i, _ := strconv.Atoi(iid)
			prop = []interface{}{s, i}
			miot = 1 //true
		} else {
			prop = []interface{}{key}
			miot = 0 //false
		}
		if value == "" {
			setp = false
		} else if setp {
			prop = append(prop, stringOrValue(value))
		}
		props = append(props, prop)
	}

	if !setp && miot == 1 && argc > 0 {
		var args []interface{}
		if arg != "#NA" {
			for _, a := range argv {
				args = append(args, stringOrValue(a))
			}
		}
		var ids []int

		for _, id := range props[0] {
			if v, ok := id.(int); ok {
				ids = append(ids, v)
			} else if v, ok := id.(string); ok {
				if v2, err := strconv.Atoi(v); err == nil {
					ids = append(ids, v2)
				}
			}
		}
		fmt.Println("MiotAction")
		return service.MiotAction(did, ids, args)
	}

	if setp {
		if miot == 1 {
			iprops := map[Iid]interface{}{}
			var pid Iid
			for i, p := range props {
				if i > 0 && len(p) == 1 {
					iprops[pid] = p[0]
				} else {
					pid = Iid{p[0].(int), p[1].(int)}
				}
			}
			fmt.Println("miot-MiotSetProps", iprops)
			return service.MiotSetProps(did, iprops)
		} else {
			sprops := map[string]interface{}{}
			var key interface{}
			for i, p := range props {
				key = p[0]
				if i > 0 && key != nil {
					sprops[fmt.Sprintf("%v", key)] = p[0]
					key = nil
				} else {
					key = p[0]
				}
			}
			fmt.Println("HomeSetProps", sprops)
			return service.HomeSetProps(did, sprops)
		}
	} else {
		if miot == 1 {
			var iprops []Iid
			for _, p := range props {
				pid := Iid{p[0].(int), p[1].(int)}
				iprops = append(iprops, pid)
			}
			fmt.Println("miot-MiotGetProps", iprops)
			return service.MiotGetProps(did, iprops)
		} else {
			var sprops []string
			for _, p := range props {
				sprops = append(sprops, fmt.Sprintf("%v", p[0]))
			}
			fmt.Println("HomeGetProps", sprops)
			return service.HomeGetProps(did, sprops)
		}
	}

	return nil, errors.New("Unknown command: " + cmd)
}
