# xiaobot - 让小爱音箱接入 AI 大模型

​	唤醒 “小爱同学” 后，只要用你指定的提示词开头与小爱对话(需未勾选"静音小爱")，就能让你选定的 AI 大模型来生成回答，彻底摆脱原来智障的小爱！

​	Play ChatGPT with Xiaomi AI Speaker fork from [xiaobot](https://github.com/longbai/xiaobot) ( [xiaogpt](https://github.com/yihong0618/xiaogpt) and convert to Go )

业余写写golang，查到了 xiaobot 这个golang项目，改了改，还是很好用的。

我的是"小爱音箱Play增强版" :( ，建议大家还是买"小爱音箱Pro"吧。

## 修改的部份

- 去掉了诸多滚动的调试信息
- AI的调用只保留了OpenAI模式API接口（现在大多的大模型供应商都支持。请google如何用[openai api](https://platform.openai.com/account/api-keys)）。也可通过指定adapter适配文件接入其他大模型。
- 采用三方的TTS感觉都太慢，干脆去掉了。还是用音箱原生的输入输出吧。
- 完善了流式响应 (结果我的"小爱音箱Play增强版"不支持查询状态，用不了:()

## 要求

1. 要有小爱音箱，推荐Pro。
2. 要有大模型帐号，推荐豆包火山引擎大模型，在模型控制台专门创建一个“联网问答Agent”，效果非常好，就是收费太贵:(-)

## 准备

1. 下载xiaobot代码
进入到%GOPATH%的src目录下：
```
   git clone https://github.com/axcom/xiaobot.git
```
项目以本地代码方式引用了ninego项目的代码，还要手动下载ninego代码：
```
   git clone https://github.com/axcom/ninego.git
```
编译程序文件：
```
   cd xiaobot
   go build cmd/xiaobot.go
```

2. 配置config.json

​	在xiaobot目录，执行以下命令:

```
   go run cmd/xiaobot.go --config
```
   执行该命令后，系统会自动弹出浏览器并跳转至地址

```
   127.0.0.1:9997/config
```

​	在该 “XiaoBot 配置中心” 页面完成参数设置即可。
​	输入完小米帐号及密码后，可以点击【获取音箱信息】按钮，选择你要连接的小米音箱。选择你的音箱后，会自动填充该音箱的“硬件类型”和“小米设备DID”信息。
​	最后点击【保存配置】即可进入测试环节：

- 保存配置成功后，等待 2~3 秒，页面会自动跳转至 xiaobot 的 Chat 对话页面。
- 在 Chat 页面发送消息，若能同时看到文字回复并听到小爱音箱的语音答复，说明 xiaobot 支持你的音箱；若失败，切换到 CMD 控制台窗口，查看具体报错信息以定位问题。

> 注意：xiaobot 服务运行期间，在 CMD 控制台窗口按 `Ctrl+C` 键可随时终止程序、退出服务。



xiaobot 提供两种连接模式，适用于不同使用场景，可根据需求选择：

### 1. 轮询模式（默认模式）

- **工作原理**：开启后，xiaobot 每间隔 1 秒查询一次小爱音箱是否有新的对话信息，一旦检测到提问，立即将问题交由 AI 大模型生成回答。
- **优点**：无需复杂设置，操作简单，适合不熟悉电脑操作的用户。
- **缺点**：因轮询存在 1 秒间隔，可能出现 “小爱先抢答一半，AI 才开始答复” 的情况，交互连贯性稍弱。

### 2. 监控模式（精准拦截模式）

- **开启方式**：通过 `-t` 参数启动，命令为 `xiaobot.exe -t`。

- **工作原理**：该模式下，xiaobot 不主动查询音箱对话，而是依赖 `mac_monitor.sh` 脚本监控音箱的网络数据（需通过 WiFi 接入或 ARP 欺骗获取数据），一旦检测到提问，立即拦截小爱原生答复并触发 AI 回答。

- **优点**：可实时阻止小爱抢答，AI 回答的连贯性极强。

- **缺点**：仅支持文字交互，无法使用小爱音箱的音乐、故事等音频功能；配置流程较复杂，需参考配套文档 “README -t 监控模式.md” 操作。

  

### 配置注意事项

#### A. API-Url 格式要求

API-Url 无需携带 `/chat/completions` 后缀，若 API 本身不需要该后缀，可直接以 `/` 结尾（如https://api.example.com/）。

#### B. 调用非 OpenAI 格式的模型（如 ollama）

xiaobot 默认支持 OpenAI 格式的模型，若需调用 ollama、anythingLLM 等其他格式模型，需按以下规则配置：

- **“AI 模型” 参数格式**

  ```
  llm服务类型=模型名称
  ```

  示例：调用本地 ollama 服务的 Qwen2.5:1.5B 模型，需填写：

  ```
  ollama=qwen2.5:1.5b
  ```

- **依赖脚本**：xiaobot 通过 “模型转换器脚本” 实现非 OpenAI 格式模型的调用，目前已提供 `ollama.adapter` 和 `anythingllm.adapter` 两种脚本（位于 xiaobot 目录）。

- **自定义脚本**：若需支持其他模型，可参考 `example.adapter` 中的提示词，让 AI 生成对应的转换脚本；注意：脚本文件名需与 “AI 模型” 参数中填写的 “llm 服务类型” 一致（如脚本名为 `custom.adapter`，则参数需填 `custom=模型名`）。

#### C. 接入字节火山引擎 “联网问答 Agent” 示例

若需接入火山引擎的智能体，可在 “GPT 选项” 中填写以下 JSON 配置（需替换为你的实际信息）：

```json
{
    "bot_id": "你的智能体ID号",
    "thinking": {
      "type": "disabled"
    }
}
```

#### D. 其它型号小爱音箱的接入

- 可以自行配置适合你的小爱音箱的控制指令
- 目前默认已配置的如下，没有的可以自行创建xiaomi.ini文件，按以下格式添到文中即可：

```json
//hardware: (tts_command, action_command, wakeup_command, playing_state[siid,piid,value], 使用play_musci接口的设备)
"OH2":   "5-3", "5-4", "5-1", "3,1,1", "1" 		// Xiaomi 智能音箱 Pro
"OH2P":  "7-3", "7-4", "7-1", "",      "1"      // Xiaomi 智能音箱 Pro          
"LX06":  "5-1", "5-5", "5-3", "",      ""       // 小爱音箱 Pro                 
"S12":   "5-1", "5-5", "5-3", "",      ""       // 小米 AI 音箱                 
"S12A":  "5-1", "5-5", "5-3", "",      ""       // 小米 AI 音箱                 
"L15A":  "7-3", "7-4", "7-1", "",      ""       // 小米 AI 音箱（第二代）            
"LX5A":  "5-1", "5-5", "5-3", "",      ""       // 小爱红外版                    
"LX05A": "5-1", "5-5", "5-3", "",      ""       // 小爱红外版                    
"LX05":  "5-1", "5-4", "5-3", "",      "1"      // 小米小爱音箱Play（2019 款）       
"X10A":  "7-3", "7-4", "7-1", "",      "1"      // 小米智能家庭屏10                
"L17A":  "7-3", "7-4", "7-1", "",      ""       // Xiaomi Sound Pro         
                                                                              
"L06A":  "5-1", "5-5", "5-2", "",      "1"      // 小爱音箱                     
"LX01":  "5-1", "5-5", "5-2", "",      ""       // 小爱音箱 mini                
"L05B":  "5-3", "5-4", "5-1", "",      "1"      // 小爱音箱 Play                
"L05C":  "5-3", "5-4", "5-1", "3,1,1", "1" 		// 小米小爱音箱Play增强版            
"L09A":  "3-1", "3-5", "3-2", "2,1,1", ""  		// 小爱音箱 Art                 
"LX04":  "5-1", "5-4", "5-2", "",      "1"      // 小爱触屏音箱                   
"ASX4B": "5-3", "5-4", "5-1", "",      "1"      // Xiaomi 智能家庭屏 Mini        
"X6A":   "7-3", "7-4", "7-1", "",      "1"      // 小米智能家庭屏6                 
"X08E":  "7-3", "7-4", "7-1", "",      "1"      // Redmi 小爱触屏音箱 Pro 8 英寸    
"L07A":  "5-1", "5-5", "5-3", "",      ""       // Redmi小爱音箱Play(l7a)       
```

以上有的型号要修正指令也可以写入ini以更正。可以在 https://home.miot-spec.com/ 查找对应音箱型号的指令
```json
tts_command     文本播放指令siid-piid
action_command  控制指令siid-piid
wakeup_command  换醒指令siid-piid
playing_state[siid-piid-value]  查询play状态(第3个value是指在播放中的状态值)
use_play_musci  是否使用play_musci接口(1=使用)
```


## xiaobot 的附加功能（小爱玩具）

作为小玩具使用，可以不用配置大模型也能玩:)。
在配置中心页面（http://127.0.0.1:9997/）的 Chat 页面左上角，有一个圆形菜单按钮，可切换至【Chat】【配置】【脚本编辑】三个页面，支持更多个性化操作：

### 1. Chat 对话功能

- 未配置 “API-Url” 时，对话会默认调用小爱原生回答，可能出现 “我要再学习” 等无效回复。
- 在 Chat 页面的【更多】菜单中，若未勾选【播放问题】和【播放回答】，对话仅在页面显示文字，不通过小爱音箱发声，相当于直接与 AI 大模型对话。

### 2. 提问内容执行脚本（query.bot）

- **脚本位置**：xiaobot 目录下的 `query.bot`（JS 脚本）。

- 工作逻辑

  ：每当 xiaobot 接收到小爱音箱的提问，会自动调用该脚本：

  - 若脚本中无 `handled` 字样：脚本在后台异步执行，不影响 AI 回答流程。

  - 若脚本中有 `handled` 字样：会中断 xiaobot 流程，先执行脚本，再根据 `handled` 的值判断是否继续：

    - `handled=true`：不再触发 AI 回答流程。
    - `handled=false`：继续执行 AI 回答流程。
  
- **脚本编写**：详见配套文档 “js 脚本引擎.md”。

### 3. 接收任务执行脚本（自定义任务）

- **功能说明**：可创建多个 JS 脚本，让外部程序通过 API 调用 xiaobot 执行任务。

- **脚本命名规则**：脚本需保存在 xiaobot 目录，文件名格式为 “任务名.bot”（如 `welcome.bot`）。

- 外部调用方式

  API 地址格式为

  ```
  http://xiaobot服务IP:端口号/task/任务名
  ```

  示例：若 xiaobot 运行在 IP 为 192.168.1.111、端口 9997 的机器上，调用`welcome.bot`脚本的地址为:

  ```
  http://192.168.1.111:9997/task/welcome
  ```

- **脚本编写**：详见配套文档 “js 脚本引擎.md”。

### 4. 音乐播放功能

- 本地音乐目录，除了可以在 “XiaoBot 配置中心” 页面设置外，也可以通过环境变量`MUSIC_PATH`来指定。
- 最好将xiaobot部署在与小爱音箱相同的局域网内。小爱音箱通过http能访问到xiaobot建立的本地音乐库。
- 若提示 “音乐http服务未能开启”，可以加 -w 参数指定你的xiaobot部署地址。如：
  ```
  xiaobot -w 192.168.1.8:9997
  ```
- xiaobot自身目前支持mp3/wav音乐文件，播放其他格式音乐需安装 FFmpeg 工具。安装后通过 FFMPEG_PATH 环境变量指定 ffprobe 的目录。

### 5. 定时任务功能

- 定时执行你编写的js脚本
- **脚本编写**：详见配套文档 “js 脚本引擎.md”。


## 感谢

- [xiaobot](https://github.com/longbai/xiaobot)
- [yihong0618](https://github.com/yihong0618)
- [xiaomi](https://www.mi.com/)
- [openai](https://openai.com/)



## 备注

因为作者只有小爱音箱Play增强版，其他的没办法测试。你的音箱若不是这个型号，可以加入QQ群:582479960讨论。