// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
package xiaobot

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"ninego/log"
	"xiaobot/jarvis"
	"xiaobot/jsengine"

	"github.com/texttheater/golang-levenshtein/levenshtein"
)

// 常量定义集中管理
const (
	maxttsWord = 256 //每段播放最大字数
)

type MiTalk struct {
	config *Config
	Box    *XiaoMi
	Bot    *MiBot

	InConversation   bool
	LastDelaySeconds int64 //末次TTS输出文字时长(用于monitor触发时判断是否立即静音)

	ChatDontTTS  int                 //0=非chat模式 1=chat不播放 -1=chat要播放
	StreamWriter http.ResponseWriter //chat模式下的流式输出

	stopchannel *Channel //用于优雅退出
}

func NewMiTalk(config *Config) *MiTalk {
	return &MiTalk{
		config:           config,
		InConversation:   false,
		LastDelaySeconds: 0, // 末次TTS输出文字时长(s秒)
		ChatDontTTS:      0,
		stopchannel:      NewChannel(),
	}
}

func (mt *MiTalk) terminated() bool {
	return mt.stopchannel.IsClosed()
}

func (mt *MiTalk) Terminate() {
	mt.stopchannel.Close() // 其内已有判断是否closed
}

func queryIn(q string, keywords []string) bool {
	if len(keywords) == 0 {
		return false
	}
	for _, k := range keywords {
		if strings.HasPrefix(q, k) {
			return true
		}
	}
	return false
}

func (mt *MiTalk) needAskJarvis(query string) bool {
	return (mt.InConversation && !strings.HasPrefix(query, WakeupKeyword)) || queryIn(query, mt.config.Keywords)
}

func (mt *MiTalk) needChangePrompt(query string) bool {
	return queryIn(query, mt.config.ChangePromptKeywords)
}

func (mt *MiTalk) changePrompt(newPrompt string) {
	for _, kw := range mt.config.ChangePromptKeywords {
		newPrompt = strings.TrimPrefix(newPrompt, kw)
	}
	newPrompt = strings.TrimSpace(newPrompt)
	if newPrompt == "" {
		return
	}

	log.Printf("Prompt changed from %q to %q\n", mt.config.Prompt, newPrompt)
	mt.config.Prompt = newPrompt
	mt.Bot.assistant.SetPrompt(newPrompt)
}

func (mt *MiTalk) askJarvis(query, answer string, chatStream bool) (string, string, error) {
	cancel := false
	go func() {
		select {
		case <-mt.stopchannel.C:
			cancel = true
		}
	}()

	if !chatStream {
		rets, err := mt.Bot.assistant.Ask(query, answer)
		if cancel {
			return "", "", nil //errors.New("cancel")
		}
		return rets, "", err
	}

	//流式返回
	strtts := "" //全部已经播放的串
	stream, err := mt.Bot.assistant.AskStream(query, answer)
	if err != nil {
		return "", strtts, err
	}
	defer stream.Close()

	if cancel {
		return "", strtts, nil //errors.New("cancel")
	}

	message := "" //尚未TTS播放的串
	mutex := &sync.Mutex{}
	ch := make(chan bool)

	runing := true
	if mt.ChatDontTTS != 1 { //chat模式1为不发音
		go func() {
			defer func() { ch <- true }()
			for {
				if mt.terminated() {
					break
				}
				if !runing && utf8.RuneCountInString(message) <= maxttsWord {
					break
				}
				if message == "" {
					continue
				}
				if i := possentence(message); i > -1 {
					str := substr(message, 0, i+1)
					n := utf8.RuneCountInString(str)
					if mt.Bot.speaker.Status() != 0 { //mt.loopStopSpeaker() {
						if n > 13 || []rune(str)[n-1] <= 13 { //长度>13或是回车
							mt.Bot.activeSpeakerVoice(0)
						} else {
							continue
						}
					}
					mutex.Lock()
					delstr(&message, 0, i+1)
					mutex.Unlock()
					strtts += str
					if strings.TrimSpace(str) != "" {
						mt.miTTS(str, true)
					}
				}
			}
		}()
	}

	log.Printf("-Ai-的回答: ")
	for {
		if cancel {
			break
		}
		receivedResponse, streamErr := stream.Recv()
		if streamErr != nil {
			err = streamErr
			break
		}
		if len(receivedResponse.Choices) > 0 {
			str := receivedResponse.Choices[0].Delta.Content
			mutex.Lock()
			message += str
			mutex.Unlock()
			fmt.Print(str) //连续输出
			if mt.ChatDontTTS != 0 && mt.StreamWriter != nil {
				// 按SSE格式发送数据
				fmt.Fprint(mt.StreamWriter, str) //fmt.Fprintf(w, "data: %s\n\n", chunk)
				// 刷新缓冲区，确保数据被立即发送
				if flusher, ok := mt.StreamWriter.(http.Flusher); ok {
					flusher.Flush()
				} else {
					http.Error(mt.StreamWriter, "Streaming not supported", http.StatusInternalServerError)
					return message, strtts, fmt.Errorf("Streaming not supported %v", http.StatusInternalServerError)
				}
			}
		}
	}
	if mt.ChatDontTTS != 0 {
		// 发送结束标记
		fmt.Fprintf(mt.StreamWriter, "data: [DONE]\n\n")
		if flusher, ok := mt.StreamWriter.(http.Flusher); ok {
			flusher.Flush()
		}
	}
	if mt.ChatDontTTS != 1 { //chat模式1为不发音
		runing = false
		<-ch //等待播放完毕
	}
	return message, strtts, err

}

// sleep 带取消功能的睡眠
func (mt *MiTalk) sleep(seconds float64) bool {
	if seconds <= 0 {
		return true
	}

	duration := time.Duration(seconds * float64(time.Second))
	select {
	case <-mt.stopchannel.C:
		return false
	case <-time.After(duration):
		return true
	}
}

// miTTS 文本转语音，优化分片处理
func (mt *MiTalk) miTTS(message string, waitForFinish bool) error {
	if mt.ChatDontTTS == 1 { // chat模式为不发音
		return nil
	}
	//return nil
	if mt.terminated() {
		return nil
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}
	mt.LastDelaySeconds = int64(calculateTtsElapse(message))
	mt.Bot.updateLatestAskTime()
	mt.Bot.activeSpeakerVoice(0)
	var value string
	for message != "" {
		if utf8.RuneCountInString(message) > maxttsWord {
			if i := possentence(message); i > -1 {
				value = substr(message, 0, i+1)
				delstr(&message, 0, i+1)
			} else {
				value = substr(message, 0, maxttsWord)
				delstr(&message, 0, maxttsWord)
			}
		} else {
			value = message
			message = ""
		}

		if err := mt.Box.MiTTS(value); err != nil {
			return fmt.Errorf("TTS命令执行失败: %w", err)
		}

		if waitForFinish || message != "" {
			statusPlaying, _ := mt.Box.SpeakerIsPlaying()
			//不支持查询状态的音箱走sleep延时(按文字长度计算延时的时间)
			if statusPlaying != 0 && statusPlaying != 1 {
				elapse := calculateTtsElapse(value)
				//startTime := time.Now() // 获取开始时间
				//fmt.Println(startTime)
				mt.sleep(float64(elapse)) //time.Sleep(elapse)
				//elapsedTime := time.Since(startTime) // 计算耗时
				//fmt.Println("执行时间：", elapsedTime)
			}
			mt.Box.WaitForTTSFinish()
		}
	}
	return nil
}

func (mt *MiTalk) miPlay(url string, waitForFinish bool) error {
	if mt.terminated() {
		return nil
	}
	mt.Bot.activeSpeakerVoice(0)
	if err := mt.Box.MiPlay(url); err != nil {
		log.Error(fmt.Sprintf("play url %v, Error: %v\n", url, err))
		return err
	}
	if waitForFinish {
		//mt.sleep(?)
		mt.Box.WaitForTTSFinish()
	}
	return nil
}

// 播放小爱的回答
func (mt *MiTalk) playMiAnswer(record *Record, waitForFinish bool) {
	if mt.ChatDontTTS == 1 { // chat模式静音
		return
	}
	if mt.terminated() {
		return
	}
	mt.Bot.activeSpeakerVoice(0)
	for _, a := range record.Answers {
		switch a.Type {
		case "TTS", "LLM":
			if err := mt.miTTS(a.Tts.Text, waitForFinish); err != nil {
				log.Error("播放小爱TTS失败:", err)
			}
			/*case "AUDIO":
			for _, b := range a.Audio.AudioInfoList {
				if err := mt.miPlay(b.Title); err != nil {
					log.Error("播放小爱音频失败:", err)
				}
			}*/
		}
	}
}

// 处理对话控制命令（开始/结束对话）
func (mt *MiTalk) handleConversationControl(query string) bool {
	if mt.InConversation {
		if queryIn(query, mt.config.EndConversation) {
			log.Println("结束对话")
			mt.Box.StopSpeaker()
			log.Println("恢复bot人设")
			mt.changePrompt(mt.Bot.Prompt)
			mt.InConversation = false
			*mt.Bot.assistant.GetHistory() = nil
			return true
		}
		if query == WakeupKeyword {
			log.Println("唤醒词，跳过处理")
			return true
		}
	} else if queryIn(query, mt.config.StartConversation) {
		log.Println("开始对话")
		mt.Bot.startSpeakerMuteLoop()
		mt.InConversation = true
		*mt.Bot.assistant.GetHistory() = make([]jarvis.RoleContent, 0)
		return false
	}
	return false
}

// 处理提示词变更
func (mt *MiTalk) handlePromptChange(query string) {
	log.Debug("需要改变bot人设提示语：", query)
	mt.Box.StopSpeaker()
	if !mt.InConversation {
		mt.Bot.Prompt = query
	}
	mt.changePrompt(query)
	mt.miTTS("好的", true)
}

// 等待完整的回答
func (mt *MiTalk) waitForCompleteAnswer(query string, answer *string) error {
	if *answer != "" {
		return nil
	}

	step := 4              //defaultSleepStep
	maxRetries := 5 * step //maxWaitSeconds
	for i := 0; i < maxRetries; i++ {
		rec, err := mt.Box.getLatestAsk()
		if err != nil {
			return fmt.Errorf("获取最新回答失败: %w", err)
		}

		if len(rec.Records) > 0 && rec.Records[0].Query == query {
			if len(rec.Records[0].Answers) > 0 {
				*answer = mt.extractAnswers(&rec.Records[0])
				/**answer = ""
				for _, a := range rec.Records[0].Answers {
					*answer += a.Tts.Text
				}*/
				return nil
			}
		} else {
			return nil // 问题已变更，无需继续等待
		}

		if !mt.sleep(1.0 / float64(step)) {
			return nil
		}
	}

	*answer = "-"
	return nil
}

// 判断是否需要停止播放
func needStop(answer string) bool {
	if answer == "" || answer == "-" {
		return true
	}

	stopPhrases := []string{
		"被你问住了", "把我难住了", "被难住了", "我好像还不太知道",
		"我暂时还回答不上", "我暂时还不支持", "暂不支持",
		"我还在研究中", "我还在学习中", "要再学习", "要更努力学习",
		"换个方式再说一遍", "换个话题", "这个话题我不太擅长",
		"最新小爱音箱APP", "本设备暂不支持该功能", "绑定音乐账号",
	}

	for _, phrase := range stopPhrases {
		if strings.Contains(answer, phrase) {
			return true
		}
	}

	return false
}

// 提取回答内容
func (mt *MiTalk) extractAnswers(record *Record) string {
	var Answer string
	if len(record.Answers) != 0 {
		for _, a := range record.Answers {
			Answer += a.Tts.Text
		}
	}
	return Answer
}

func (mt *MiTalk) playThinking() {
	// 播放思考提示（此后loopStopSpeaker状态=False？）
	if mt.ChatDontTTS != 0 {
		return
	}

	// []string{"让我先想想", "让我想一下", "让我想一想"}
	n := len(mt.config.Thinkingwords)
	if n == 0 {
		return
	}
	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(n + 2)
	if i >= n {
		i = 0
	}
	mithinking := mt.config.Thinkingwords[i]

	if mithinking != "" && mt.Bot.loopStopSpeaker() {
		//mt.wg.Add(1)
		go func() {
			//time.Sleep(time.Duration(0.6 * float64(time.Second)))
			if !mt.sleep(0.6) {
				return
			}
			if mt.Bot.loopStopSpeaker() && !mt.terminated() {
				// 后台播放（解除静音）
				mt.miTTS(mithinking, false)
				// 计算播计等待时长
				elapse := -1 - int(calculateTtsElapse(mithinking)) // mt.loopStopSpeaker(-1) 已播放think
				mt.Bot.loopStopSpeaker(elapse)
				//延时等待think播放完毕
				//time.Sleep((elapse + 1) * time.Second)
				for i := elapse; i < -1; i++ {
					if !mt.sleep(1) {
						return
					}
					if mt.Bot.speaker.Status() == i {
						mt.Bot.loopStopSpeaker(i - 1)
					} else {
						return
					}
				}
			}
		}()
	}
}

// 统一处理对话逻辑，消除代码冗余
func (mt *MiTalk) handleConversation(record Record, isMuteMode bool) error {
	// 问题
	query := strings.TrimSpace(record.Query)
	if query == "" {
		return nil
	}
	// Leave weather, music and home-control commands entirely to native XiaoAI.
	explicitAI := queryIn(query, mt.config.Keywords)
	if !mt.InConversation && !explicitAI && !mt.needChangePrompt(query) && !queryIn(query, mt.config.StartConversation) {
		return nil
	}
	// 问题脚本处理
	if mt.config.QueryJS != "" {
		log.Println("Call query.bot")
		// 检查代码中是否包含handled变量的直接引用
		if !strings.Contains(mt.config.QueryJS, "handled") {
			go func() { jsengine.Exec_queryJS(query, mt.config.QueryJS) }()
		} else {
			if handled, err := jsengine.Exec_queryJS(query, mt.config.QueryJS); handled {
				return err
			}
		}
	}

	// 提取初始小爱的回答
	answer = mt.extractAnswers(&record)

	// 没有配置AI帐号
	hasGPT := mt.Bot.HasGPT

	log.Println(strings.Repeat("-", 20))
	log.Printf("问题：%s？\n", query)

	// 根据模式决定是否立即静音
	firstlyStopped := needStop(answer)
	if explicitAI || mt.InConversation || isMuteMode || firstlyStopped || mt.Bot.monitor.Status() != 0xFFFF /*监控模式*/ {
		if !hasGPT && mt.ChatDontTTS == -1 {
			// 无AI配置,且chat要发音
		} else {
			mt.Bot.startSpeakerMuteLoop()
		}
	}

	//mt.wg.Add(1)
	go func() {
		defer func() {
			defer mt.Terminate()
			if r := recover(); r != nil {
				log.Error("Recovered from panic:", r)
			}

			//isclosed := mt.terminated()
			//mt.wg.Done()

			// 更新对话历史
			if mt.InConversation {
				log.Printf("继续对话, 或用`%s`结束对话\n", mt.config.EndConversation)
				history := mt.Bot.assistant.GetHistory()
				*history = append(*history,
					jarvis.RoleContent{Role: "user", Content: query},
					jarvis.RoleContent{Role: "assistant", Content: answer},
				)
				if !mt.terminated() /*!isclosed*/ && query != WakeupKeyword && mt.ChatDontTTS == 0 {
					mt.Box.WakeUp()
				}
			}
		}()

		// 处理对话控制命令
		if mt.handleConversationControl(query) {
			return
		}

		// 处理提示词变更
		if mt.needChangePrompt(query) {
			mt.handlePromptChange(query)
			return
		}

		// 在非对话模式且没Loop静音下检查是否需要处理
		if /*!mt.InConversation &&*/ !firstlyStopped && !mt.Bot.loopStopSpeaker() && !mt.needAskJarvis(query) {
			//未被静音(已一定不是对话模式||监控模式),提问词首不匹配+且为有效回答!firstlyStopped(=!needStop)
			log.Println("不需要问AI:", answer)
			return
		}

		// 后台播放思考提示（此后loopStopSpeaker状态=False？）
		if hasGPT {
			mt.playThinking()
		}

		// 非对话且非静音小爱模式下等待小爱完整回答
		if (!explicitAI && !mt.InConversation && !isMuteMode && answer == "") || !hasGPT {
			if err := mt.waitForCompleteAnswer(query, &answer); err != nil {
				log.Error("等待完整回答失败:", err)
			}
			if !hasGPT {
				return
			}
		}

		log.Printf("小爱的回答: %s\n", answer)

		// 调用AI获取回答
		unknown := ""
		if !explicitAI && !mt.InConversation && !needStop(answer) { // Explicit questions always receive a fresh LLM answer.
			unknown = answer
		}
		message, aiSpokenText, err := mt.askJarvis(query, unknown, mt.config.Stream)
		if err != nil {
			if errors.Is(err, io.EOF /*流式响应*/) {
				log.Println(message)
			} else {
				log.Error("AskGPT error:", err)
				return
			}
		} else {
			log.Printf("-AI-的回答: %s\n", message)
		}

		// 检查回答相似度，避免重复播放
		fullAiResponse := aiSpokenText + message
		needAI := (answer != fullAiResponse && levenshtein.RatioForStrings([]rune(answer), []rune(fullAiResponse), levenshtein.DefaultOptions) < 0.5)
		if explicitAI || mt.InConversation || isMuteMode || firstlyStopped || mt.Bot.monitor.Status() != 0xFFFF /*监控模式*/ ||
			needAI {
			// 播放AI的回答
			mt.miTTS(message, mt.InConversation /*false*/)
			//log.Println("ai-tts播放:", message)
		} else if mt.Bot.speaker.Status() != 0 { //mt.loopStopSpeaker() {
			// 播放小爱的回答
			mt.playMiAnswer(&record, mt.InConversation)
			//log.Println("xiaoai播放:", answer)
		} else {
			//log.Println("no speaker", mt.InConversation, isMuteMode, firstlyStopped, needAI, mt.Bot.speaker.Status())
		}
		answer = fullAiResponse
		return
	}()

	return nil
}

var answer string //设为全局可供chat使用
