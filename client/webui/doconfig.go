package webui

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"ninego/log"
	"xiaobot"
	"xiaobot/miservice"
)

var submit = false

func do_setConfig(writer http.ResponseWriter, request *http.Request) {
	log.Debug("config submit", request.Method, request.Body)
	if request.Method != http.MethodPost || request.Body == nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	if submit {
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	log.Debug("body->config", json.NewDecoder(request.Body))
	err := json.NewDecoder(request.Body).Decode(&config)
	if err != nil {
		log.Error("Error:", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	log.Debug("", config)
	// 简单的验证
	if config.Account == "" {
		http.Error(writer, "missing xiaomi account field", http.StatusBadRequest)
		return
	}

	// 保存配置到文件
	err = saveConfig(config)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Debug("Received config:", config)

	// 动态更新Music目录
	MusicFS.UpdateHandler(config.MusicPath)
	log.Debug("Music folder:", config.MusicPath)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	s := "配置写入config.json完成，请运行 'xiaobot' 启动bot服务。"
	writer.Write([]byte(`{"message":"` + s + `"}`))
	log.Println("配置写入config.json")

	// 通知主程序关闭完成
	WebDone <- true
}

// 保存配置到文件
func saveConfig(config *xiaobot.Config) error {
	// 将配置转换为JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// 确定配置文件路径
	configPath := xiaobot.ConfigFile
	if configPath == "" {
		configPath = filepath.Join("config.json")
	}

	// 写入文件
	return ioutil.WriteFile(configPath, data, 0600)
}

//获取配置的处理函数
func do_getConfig(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 确定配置文件路径
	// 读取已保存的配置文件
	configPath := xiaobot.ConfigFile
	if configPath == "" {
		configPath = filepath.Join("config.json")
	}
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		// 如果文件不存在，返回空配置
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("{}"))
		return
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Error("Error parsing config:", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(config)
}

type device struct {
	Name     string `json:"name"`
	MiotDID  string `json:"midid"`
	Hardware string `json:"hardware"`
}
type account struct {
	UserID   string `json:"account"`
	Password string `json:"password"`
}

func do_getDeviceList(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	miuser := account{}
	err := json.NewDecoder(request.Body).Decode(&miuser)
	if err != nil {
		log.Error("Error decoding request body:", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	account := miservice.NewAccount(
		miuser.UserID,
		miuser.Password,
		miservice.NewTokenStore(filepath.Join(os.Getenv("HOME"), ".mi.token")),
	)
	service := miservice.NewAIService(account)
	deviceList, err := service.DeviceList(0)
	if err != nil {
		log.Error("Error getting device list:", err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	var devices []device
	for i := 0; i < len(deviceList); i++ {
		// 查重
		finded := false
		for _, data := range devices {
			if data.MiotDID == deviceList[i].MiotDID {
				finded = true
				break
			}
		}
		if finded {
			continue
		}
		data := device{
			Name:     deviceList[i].Alias,
			MiotDID:  deviceList[i].MiotDID,
			Hardware: deviceList[i].Hardware,
		}

		// 是否支持
		finded = false
		for k, _ := range xiaobot.HardwareCommandDict {
			if k == data.Hardware {
				finded = true
				break
			}
		}
		if !finded {
			data.Hardware += "(暂不支持)"
		}

		devices = append(devices, data)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	// 直接返回设备列表，不进行二次编码
	json.NewEncoder(writer).Encode(devices)
}
