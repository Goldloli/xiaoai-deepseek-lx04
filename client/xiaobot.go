// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
package xiaobot

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"ninego/log"
	"xiaobot/jarvis"
	"xiaobot/jsengine"
)

const (
	//SpeakerThinking = -1 // 正在播放思考提示
	SpeakerActive = 0 // 可正常播放
	SpeakerMuted  = 1 // 静音状态
)

type MiBot struct {
	config *Config
	Box    *XiaoMi
	Talk   *MiTalk

	records       chan Record
	LastTimestamp int64 //末次对话时间戳

	HasGPT    bool
	assistant jarvis.Jarvis
	Prompt    string //对话前bot人设

	monitor *StateStore //0=未轮询 >0轮询中 0xFFFF为非监控模式
	speaker *StateStore //0=未静音 1=静音 -1=已播think 小于-1=正在播thing
}

func NewMiBot(config *Config) *MiBot {
	bot := &MiBot{
		config:        config,
		Box:           NewXiaoMi(config),
		Talk:          NewMiTalk(config),
		records:       make(chan Record, 100),
		HasGPT:        false,
		LastTimestamp: 0, // 启动首次不动作
		monitor:       NewState(0),
		speaker:       NewState(0),
	}
	bot.Talk.Box = bot.Box
	bot.Talk.Bot = bot
	bot.Talk.Terminate()
	return bot
}

// UpdateWorking 更新工作状态
func (mt *MiBot) UpdateMonitor(step int) {
	if mt.monitor.Status() == 0xFFFF { //非监控模式
		return
	}
	mt.monitor.UpdateFor(func() {
		if step > 0 {
			log.Println("monitoring triggered", mt.monitor.Status(), mt.Talk.terminated())
			// 未工作立即静音
			if mt.monitor.Data == 0 && mt.Talk.terminated() {
				mt.activeSpeakerVoice() //loopStopSpeaker(0)
				//在上一TTS播完后才stop
				if time.Now().Unix()*1000 >= mt.LastTimestamp+(mt.Talk.LastDelaySeconds*1000) {
					//log.Println("startLoopMuteSpeaker")
					mt.startSpeakerMuteLoop()
				}
			}
			// 当前时间减2秒(触发点在wss之后/音箱回应时?)
			mt.updateLatestAskTime(-2)
			mt.monitor.Data += step // 加计数
		} else if mt.monitor.Data > 0 {
			mt.monitor.Data += step // 减计数
		}
	})
}

// 更新LastTimestamp：默认当前时间减1秒
func (mt *MiBot) updateLatestAskTime(offsetSeconds ...int) {
	offset := -1 // 默认当前时间减1秒
	if len(offsetSeconds) > 0 {
		offset = offsetSeconds[0] // +-偏移
	}

	if offset == 0 { // 为0时取末次对话时间
		records, err := mt.Box.getLatestAsk()
		if err == nil && len(records.Records) > 0 {
			mt.LastTimestamp = records.Records[0].Time
			return
		}
	}

	mt.LastTimestamp = int64((time.Now().Unix() + int64(offset)) * 1000)
}

// 设置loopStopSpeaker标志: 0=未静音 1=静音 -1=已播think 小于-1=正在播thing
// 返回是否正在静音中
func (mt *MiBot) loopStopSpeaker(flag ...int) bool {
	//return mt.speaker == 1
	return mt.speaker.UpdateFor(func() {
		if len(flag) > 0 {
			//log.Println("loopStopSpeaker=", flag[0])
			switch flag[0] {
			case 1:
				mt.speaker.Data = SpeakerMuted
			case 0:
				loop := -1 * (mt.speaker.Data + 1)
				if loop > 0 && !mt.Talk.terminated() {
					start := time.Now()
					for mt.speaker.Data < -1 && time.Since(start) < time.Duration(loop)*time.Second {
						if !mt.Talk.sleep(1) {
							return
						}
						//time.Sleep(time.Second)
						mt.speaker.Data++
					}
				}
				/*for mt.speaker.Data < -1 {
					time.Sleep(time.Second)
					mt.speaker.Data++
				}*/
				mt.speaker.Data = 0
			default:
				mt.speaker.Data = flag[0]
			}
		}
	}) == 1
}

// 启动停止播放的循环
func (mt *MiBot) startSpeakerMuteLoop() {
	if mt.loopStopSpeaker() {
		return
	}
	mt.loopStopSpeaker(SpeakerMuted)
	mt.Box.StopSpeaker() //立即执行静音
	//mt.wg.Add(1)
	go func() {
		//defer mt.wg.Done()
		for {
			if !mt.Talk.sleep(1 / 4) { //time.Sleep(time.Second / 3)
				return
			}
			if mt.Talk.terminated() {
				return
			}
			if !mt.loopStopSpeaker() {
				return
			}
			mt.Box.StopSpeaker()
		}
	}()
}

// 停止静音 Stop-MuteLoop
func (mt *MiBot) activeSpeakerVoice(flag ...int) {
	if len(flag) > 0 {
		if mt.Talk.terminated() {
			return
		}
	}
	mt.loopStopSpeaker(SpeakerActive)
}

func (mt *MiBot) pollLatestAsk() {
	for {
		if mt.monitor.Status() > 0 {
			//log.Debug("Now listening xiaoai new message timestamp: %d", mt.LastTimestamp)
			start := time.Now()
			records, err := mt.Box.getLatestAsk()
			if err != nil {
				log.Error("Error getting latest ask: ", err)
				time.Sleep(5 * time.Second)
				continue
			}

			if len(records.Records) > 0 {
				sort.Slice(records.Records, func(i, j int) bool {
					return records.Records[i].Time > records.Records[j].Time
				})
				/*for _, r := range records.Records {
					fmt.Println(r.Time, r.Query)
				}*/
				r := records.Records[0]
				if mt.LastTimestamp == 0 { // 首次不动作
					mt.LastTimestamp = r.Time
				} else if r.Time > mt.LastTimestamp {
					mt.LastTimestamp = r.Time
					mt.UpdateMonitor(-1)
					mt.records <- r //捕获到对话
					log.Debug("get latest ask:", mt.LastTimestamp, mt.records)
				}
			}

			elapsed := time.Since(start)
			if elapsed < time.Second {
				if mt.monitor.Status() != 0xFFFF { //非监控模式
					time.Sleep(time.Second / 10)
				} else {
					time.Sleep(time.Second - elapsed)
				}
			}
		} else {
			time.Sleep(time.Second / 10)
		}
	}
}

func (mt *MiBot) InitAllData() error {
	if err := mt.Box.InitMiBox(); err != nil {
		return fmt.Errorf("初始化设备失败: %w", err)
	}

	//获取bot的方法到map(供JS调用使用)
	mt.SetVM(jsengine.BotfuncMap)

	llmtype := "openai"
	model := mt.config.Bot
	if index := strings.Index(model, "="); index != -1 {
		llmtype = strings.TrimSpace(model[0:index])
		model = strings.TrimSpace(model[index+1:])
	}
	var adapter *jsengine.Program
	if llmtype != "openai" && llmtype != "" {
		if proc, err := jsengine.LoadLlmAdpater(llmtype); err != nil {
			log.Error("加载模型适配器出错：", err)
		} else {
			adapter = proc
			log.Println("加载模型适配器：", llmtype, model)
		}
	}
	mt.assistant = &jarvis.GhatGPT{
		Model:          model,
		Key:            mt.config.OpenAIKey,
		Backend:        mt.config.OpenAIBackend,
		Proxy:          mt.config.Proxy,
		Prompt:         mt.config.Prompt,
		GPTOptions:     mt.config.GPTOptions,
		Adapter:        adapter,
		HistoryMessage: make([]jarvis.RoleContent, 0),
	}
	mt.assistant.SetPrompt(mt.config.Prompt)
	mt.HasGPT = (mt.config.OpenAIBackend != "")

	return nil
}

func (mt *MiBot) startNewTalk() {
	//mt.loopStopSpeaker(0)

	// 停止上一问题处理
	InConversation := mt.Talk.InConversation
	mt.Talk.Terminate()
	// 开始新的问题处理
	newTalk := NewMiTalk(mt.config)
	newTalk.Box = mt.Box
	newTalk.Bot = mt
	newTalk.InConversation = InConversation
	// 原子替换引用
	mt.Talk = newTalk
}

// Run 运行机器人主逻辑（monitor == 0xFFFF 代表非监控模式）
func (mt *MiBot) Run(monitor int) error {
	log.Println("Running xiaobot 1.36 now")
	if mt.config.MuteXiaoAI && monitor == 0xFFFF {
		log.Printf("用`%s`开头来提问\n", strings.Join(mt.config.Keywords, "/"))
	}
	log.Printf("用`%s`开始持续对话\n", mt.config.StartConversation)
	log.Println("✅ 开启服务...")

	mt.monitor.Set(monitor)
	go mt.pollLatestAsk()
	for {
		select {
		case record := <-mt.records:
			// 停止上一问题处理
			if monitor == 0xFFFF { //轮询模式（触发模式已置0且可能为1）
				mt.activeSpeakerVoice() //loopStopSpeaker(0)
			}
			// 开始新的问题处理
			mt.startNewTalk()
			if err := mt.Talk.handleConversation(record, mt.config.MuteXiaoAI && monitor == 0xFFFF); err != nil {
				log.Error("对话处理错误:", err)
			}
		}
	}

	return nil
}

// Chat 同机器人聊天(流式返回)
func (mt *MiBot) ChatStream(query string, w http.ResponseWriter) error {
	//流式返回
	stream, err := mt.assistant.AskStream(query, "")
	if err != nil {
		http.Error(w, fmt.Sprintf("AskGPT error: %v", err), http.StatusInternalServerError)
		return err
	}
	defer stream.Close()
	for {
		receivedResponse, streamErr := stream.Recv()
		if streamErr != nil {
			err = streamErr
			break
		}
		if len(receivedResponse.Choices) == 0 {
			err = io.EOF
			break
		}
		chunk := receivedResponse.Choices[0].Delta.Content
		// 按SSE格式发送数据
		fmt.Fprint(w, chunk) //fmt.Fprintf(w, "data: %s\n\n", chunk)
		// 刷新缓冲区，确保数据被立即发送
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		} else {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return fmt.Errorf("Streaming not supported %v", http.StatusInternalServerError)
		}
	}
	if errors.Is(err, io.EOF /*流式响应*/) {
		// 发送结束标记
		fmt.Fprintf(w, "data: [DONE]\n\n")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	} else if err != nil {
		http.Error(w, fmt.Sprintf("AskGPT error: %v", err), http.StatusInternalServerError)
		return err
	}
	return nil
}

// Chat 同机器人聊天(非流式返回)
func (mt *MiBot) Chat(query string, mode int) string {
	if mode == 0 && mt.HasGPT {
		answer = ""
		message, err := mt.assistant.Ask(query, "")
		if err != nil {
			log.Error("AskGPT error:", err)
			return ""
		}
		answer = message
		return answer
	}

	//通过小爱音箱
	//defer func() { mt.chatDontTTS = 0 }()
	mt.startNewTalk()
	mt.Talk.ChatDontTTS = -1

	answer = ""
	// 播放问题
	if mode == 1 || mode == 3 {
		//mt.Talk.Terminate()
		//mt.wg.Wait()
		//mt.stopch = NewChannel()

		mt.Talk.miTTS(query, true)
		time.Sleep(1 * time.Second) //time.Sleep(calculateTtsElapse(query))
	}
	if mode <= 1 {
		mt.Talk.ChatDontTTS = 1
	}
	// 停止Speaker
	if !mt.HasGPT && mt.Talk.ChatDontTTS == -1 {
		// 无AI配置,且chat要发音
	} else {
		mt.startSpeakerMuteLoop()
	}
	// 更新LatestAskTime为当前
	mt.updateLatestAskTime() //mt.LastTimestamp = int64(time.Now().Unix() * 1000)
	// 先禁止Run那边读取对话
	originalMonitor := mt.monitor.Status() // 保存原始监控状态
	defer func() {
		mt.monitor.Set(originalMonitor) // 恢复原始监控状态
	}()
	mt.monitor.Set(0) // 禁止 pollLatestAsk
	// 提交问题
	mt.Box.MiAction(query)
	// 轮循5秒获取query
	var record Record
	for i := 0; i < 10*5; i++ {
		records, err := mt.Box.getLatestAsk()
		if err != nil {
			log.Error("Error getting latest ask: ", err)
			continue
		}
		if len(records.Records) > 0 {
			if records.Records[0].Query == query {
				record = records.Records[0]
				mt.LastTimestamp = record.Time
				break
			}
		}
		time.Sleep(time.Second / 10)
	}
	if record.Query == "" {
		return ""
	}
	//  处理对话并等待完成，获取答复
	if err := mt.Talk.handleConversation(record, false); err != nil {
		log.Error("对话处理错误:", err)
	}

	// 等待处理完成（带90s超时）
	done := make(chan struct{})
	go func() {
		for !mt.Talk.terminated() {
			time.Sleep(time.Second / 10)
		}
		close(done) // WaitGroup完成后关闭channel
	}()
	select {
	case <-done:
		// 等待要么完成要么超时
	case <-time.After((90 + calculateTtsElapse(query)) * time.Second):
		log.Println("Chat等待超时（90秒）")
		mt.Talk.Terminate()
	}
	result := answer

	// 设置LastTimestamp，避免重复触发？
	mt.updateLatestAskTime(0)
	return result
}

func (mt *MiBot) SetVM(bot map[string]interface{}) {
	bot["monitor"] = func(step int) {
	}
	bot["askAI"] = func(query string) string {
		ai := NewMiTalk(mt.config)
		ai.Box = mt.Box
		ai.Bot = mt
		message, strtts, err := ai.askJarvis(query, "", mt.config.Stream)
		if err != nil {
			log.Error("js执行aksAI出错：", err)
			return ""
		}
		return strtts + message
	}
	bot["tts"] = func(text string, wait bool) bool {
		if mt.monitor.Status() != 0xFFFF { // monitor模式
			defer mt.monitor.Set(0)
			mt.monitor.Set(0xFFFF) // 非监控模式
		}
		err := mt.Box.MiTTS(text)
		if err != nil {
			return false
		}
		if wait {
			statusPlaying, _ := mt.Box.SpeakerIsPlaying()
			//不支持查询状态的音箱走sleep延时(按文字长度计算延时的时间)
			if statusPlaying != 0 && statusPlaying != 1 {
				elapse := calculateTtsElapse(text)
				time.Sleep(elapse * time.Second)
			}
			mt.Box.WaitForTTSFinish()
		} else {
			time.Sleep(time.Duration(1.5 * float64(time.Second))) //至少延时1.5秒
		}
		return true
	}
	bot["action"] = mt.Box.MiAction
	bot["playurl"] = mt.Box.MiPlay
	bot["stopspeaker"] = mt.Box.StopSpeaker
	bot["wakeup"] = mt.Box.WakeUp
	bot["sleep"] = func(sec float64) {
		time.Sleep(time.Duration(float64(time.Second) * sec))
	}
	bot["elapsed"] = func(text string) int {
		i := calculateTtsElapse(text)
		return int(i)
	}
	bot["wait"] = mt.Box.WaitForTTSFinish
}
