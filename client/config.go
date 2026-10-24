package xiaobot

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// 配置文件
var ConfigFile string

type Config struct {
	Hardware string `json:"hardware" toml:"hardware"` //硬件类型
	Account  string `json:"account" toml:"account"`   //小米账号
	Password string `json:"password" toml:"password"` //密码
	MiDID    string `json:"mi_did" toml:"mi_did"`     //设备did

	Bot           string                 `json:"bot" toml:"bot"`                         //使用的 bot 类型，目前支持gpt3,chatgptapi和newbing
	OpenAIKey     string                 `json:"openai_key" toml:"openai_key"`           //openai的apikey
	OpenAIBackend string                 `json:"openai_backend" toml:"openai_backend"`   //请注意，此处你输入的 api 地址应该是'https://xxxx/v1'的字样
	Proxy         string                 `json:"proxy,omitempty" toml:"proxy,omitempty"` //支持 HTTP 代理，传入 http proxy URL
	GPTOptions    map[string]interface{} `json:"gpt_options" toml:"gpt_options"`         //OpenAI API 的参数字典

	Keywords             []string `json:"keyword" toml:"keyword"`                             //自定义请求词列表	["请问"]
	ChangePromptKeywords []string `json:"change_prompt_keyword" toml:"change_prompt_keyword"` //更改提示词触发列表	["更改提示词"]
	Thinkingwords        []string `json:"thinking" toml:"thinking"`                           //小爱思考说词
	Prompt               string   `json:"prompt" toml:"prompt"`                               //自定义prompt	请用100字以内回答
	MuteXiaoAI           bool     `json:"mute_xiaoai" toml:"mute_xiaoai"`                     //快速停掉小爱自己的回答，可以快速停掉小爱的回答
	UseCommand           bool     `json:"use_command" toml:"use_command"`                     //使用 MI command 与小爱交互，目前已知 LX04 和 L05B L05C 可能需要使用 true

	StartConversation []string `json:"start_conversation" toml:"start_conversation"` //开始持续对话关键词	开始持续对话
	EndConversation   []string `json:"end_conversation" toml:"end_conversation"`     //结束持续对话关键词	结束持续对话

	//使用大模型流式响应，获得更快的响应
	Stream bool `json:"stream" toml:"stream"`

	//是否打印详细日志
	Verbose bool `json:"verbose" toml:"verbose"`

	TokenPath string `json:"-" toml:"-"` //`json:"token_path" toml:"token_path"`

	MusicPath string `json:"music_path" toml:"music_path"`

	QueryJS string            `json:"-" toml:"-"`
	TaskJS  map[string]string `json:"-" toml:"-"`
}

func (c *Config) NeedSetup() bool {
	ParseHardwareCommandDict("xiaomi.ini") //初始化硬件信息

	return c.Account == "" || c.Password == "" || c.Hardware == "" || c.MiDID == ""
}

func (c *Config) PostInit() error {

	if v := os.Getenv("MI_USER"); v != "" {
		c.Account = v
	}
	if v := os.Getenv("MI_PASS"); v != "" {
		c.Password = v
	}
	if v := os.Getenv("MI_DID"); v != "" {
		c.MiDID = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		c.OpenAIKey = v
	}
	if v := os.Getenv("OPENAI_MODEL"); v != "" {
		c.Bot = v
	}
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		c.OpenAIBackend = v
	}
	if v := os.Getenv("MUSIC_PATH"); v != "" {
		c.MusicPath = v
	}
	if c.TokenPath == "" {
		c.TokenPath = filepath.Join(os.Getenv("HOME"), ".mi.token")
	}
	if len(c.Keywords) == 0 {
		c.Keywords = JarvisKeyWords
	}
	if c.Bot == "" {
		c.Bot = "gpt"
	}
	if c.QueryJS == "" {
		if bytes, err := os.ReadFile(GetExecutableDir() + "/query.bot"); err == nil {
			c.QueryJS = strings.TrimSpace(string(bytes))
		}
	}
	if c.TaskJS == nil {
		c.TaskJS = make(map[string]string)
		// 使用Glob函数匹配所有.bot文件，支持通配符
		filePaths, err := filepath.Glob(GetExecutableDir() + "/*.bot")
		if err == nil {
			// 遍历匹配到的文件
			for _, filePath := range filePaths {
				// 读取文件内容
				content, err := os.ReadFile(filePath)
				if err != nil {
					continue
				}
				// 获取文件名（不含路径）
				filename := filepath.Base(filePath)
				// 移除.bot后缀作为key
				key := strings.TrimSuffix(filename, ".bot")
				if key != "query" {
					c.TaskJS[key] = strings.TrimSpace(string(content))
				}
			}
		}
	}
	return nil
}

func NewConfigFromFile(path string) (*Config, error) {
	config := &Config{}
	if err := config.ReadFromFile(path); err != nil {
		return nil, err
	}
	return config, nil
}

func NewConfigFromOptions(options map[string]interface{}) (*Config, error) {
	config := &Config{}

	if options["config"] != nil {
		err := config.ReadFromFile(options["config"].(string))
		if err != nil {
			return nil, err
		}
	}

	for key, value := range options {
		switch key {
		case "hardware":
			config.Hardware = value.(string)
		case "account":
			config.Account = value.(string)
		case "password":
			config.Password = value.(string)
		case "openai_key":
			config.OpenAIKey = value.(string)
		case "proxy":
			config.Proxy = value.(string)
		case "mi_did":
			config.MiDID = value.(string)
		case "keyword":
			config.Keywords = strings.Split(value.(string), ",")
		case "thinking":
			config.Thinkingwords = strings.Split(value.(string), ",")
		case "change_prompt_keyword":
			config.ChangePromptKeywords = strings.Split(value.(string), ",")
		case "prompt":
			config.Prompt = value.(string)
		case "mute_xiaoai":
			config.MuteXiaoAI = value.(bool)
		case "bot":
			config.Bot = value.(string)
		case "use_command":
			config.UseCommand = value.(bool)
		case "verbose":
			config.Verbose = value.(bool)
		case "start_conversation":
			config.StartConversation = strings.Split(value.(string), ",")
		case "end_conversation":
			config.EndConversation = strings.Split(value.(string), ",")
		case "stream":
			config.Stream = value.(bool)
		case "gpt_options":
			config.GPTOptions = value.(map[string]interface{})
		case "token_path":
			//config.TokenPath = value.(string)
		case "music_path":
			config.MusicPath = value.(string)
		}
	}

	err := config.PostInit()
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (c *Config) readFromJson(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, c)
}

func (c *Config) readFromTOML(configPath string) error {
	_, err := toml.DecodeFile(configPath, c)
	return err
}

func (c *Config) ReadFromFile(configPath string) (err error) {
	ConfigFile = configPath

	if strings.HasSuffix(configPath, ".toml") {
		err = c.readFromTOML(configPath)
	} else if strings.HasSuffix(configPath, ".json") {
		err = c.readFromJson(configPath)
	} else {
		return errors.New("invalid config file type")
	}
	if err != nil {
		return err
	}
	return c.PostInit()
}

const LatestAskApi = "https://userprofile.mina.mi.com/device_profile/v2/conversation?source=dialogu&hardware={hardware}&timestamp={timestamp}&limit=2"
const WakeupKeyword = "小爱同学"
const MicoApi = "micoapi"

//不同型号使用不同的siid/piid组合控制语音功能
var HardwareCommandDict = map[string][5]string{
	//hardware: (tts_command, action_command, wakeup_command, playing_state[siid,piid,value], 使用play_musci接口的设备)
	"OH2":   {"5-3", "5-4", "5-1", "3,1,1", "1"}, // Xiaomi 智能音箱 Pro
	"OH2P":  {"7-3", "7-4", "7-1", "", "1"},      // Xiaomi 智能音箱 Pro
	"LX06":  {"5-1", "5-5", "5-3", "", ""},       // 小爱音箱 Pro
	"S12":   {"5-1", "5-5", "5-3", "", ""},       // 小米 AI 音箱
	"S12A":  {"5-1", "5-5", "5-3", "", ""},       // 小米 AI 音箱
	"L15A":  {"7-3", "7-4", "7-1", "", ""},       // 小米 AI 音箱（第二代）
	"LX5A":  {"5-1", "5-5", "5-3", "", ""},       // 小爱红外版
	"LX05A": {"5-1", "5-5", "5-3", "", ""},       // 小爱红外版
	"LX05":  {"5-1", "5-4", "5-3", "", "1"},      // 小米小爱音箱Play（2019 款）
	"X10A":  {"7-3", "7-4", "7-1", "", "1"},      // 小米智能家庭屏10
	"L17A":  {"7-3", "7-4", "7-1", "", ""},       // Xiaomi Sound Pro

	"L06A":  {"5-1", "5-5", "5-2", "", "1"},      // 小爱音箱
	"LX01":  {"5-1", "5-5", "5-2", "", ""},       // 小爱音箱 mini
	"L05B":  {"5-3", "5-4", "5-1", "", "1"},      // 小爱音箱 Play
	"L05C":  {"5-3", "5-4", "5-1", "3,1,1", "1"}, // 小米小爱音箱Play增强版
	"L09A":  {"3-1", "3-5", "3-2", "2,1,1", ""},  // 小爱音箱 Art
	"LX04":  {"5-1", "5-4", "5-2", "", "1"},      // 小爱触屏音箱
	"ASX4B": {"5-3", "5-4", "5-1", "", "1"},      // Xiaomi 智能家庭屏 Mini
	"X6A":   {"7-3", "7-4", "7-1", "", "1"},      // 小米智能家庭屏6
	"X08E":  {"7-3", "7-4", "7-1", "", "1"},      // Redmi 小爱触屏音箱 Pro 8 英寸
	"L07A":  {"5-1", "5-5", "5-3", "", ""},       // Redmi小爱音箱Play(l7a)
	// add more here https://home.miot-spec.com/
}

/*# 需要使用 play_musci 接口的设备型号
/*_USE_PLAY_MUSIC_API = [
    "LX04",
    "LX05",
    "L05B",
    "L05C",
    "L06",
    "L06A",
    "X08A",
    "X10A",
    "X08C",
    "X08E",
    "X8F",
    "X4B",
    "OH2",
    "OH2P",
    "X6A",
]*/
func ParseHardwareCommandDict(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过空行和注释行
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// 移除行尾注释
		if idx := strings.Index(line, "//"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		// 解析键值对
		l := strings.Split(line, ":")
		if len(l) != 2 {
			continue // 跳过解析失败的行
		}
		key := l[0]
		values := strings.Split(l[1], ",")
		// 转换为固定长度数组
		var attr [5]string
		for i, v := range values {
			if i >= 5 {
				break
			}
			attr[i] = v
		}
		HardwareCommandDict[key] = attr
	}

	return nil
}

var DefaultCommand = [5]string{"5-3", "5-4", "5-1", "3,1", "1"}

var JarvisKeyWords = []string{"请", "你", "帮我"}
var ChangePromptKeyWord = []string{"你是", "我是"}
var Prompt = "以下请用100字以内回答，请只回答文字不要带链接"
