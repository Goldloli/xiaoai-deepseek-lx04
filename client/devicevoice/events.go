// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
// Package devicevoice handles the LX04's local speech events without cloud polling.
package devicevoice

import (
	"encoding/json"
	"strings"
)

type Event struct{ Kind, Text string }

func ParseEvent(line string) Event {
	if i := strings.Index(line, "MSG_ASR_FINAL_RESULT:"); i >= 0 {
		return Event{"query", strings.TrimSpace(line[i+len("MSG_ASR_FINAL_RESULT:"):])}
	}
	i := strings.Index(line, "instruction=")
	if i < 0 {
		return Event{}
	}
	var item struct {
		Header struct {
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
		} `json:"header"`
		Payload struct {
			Text    string `json:"text"`
			Final   bool   `json:"is_final"`
			Results []struct {
				Text string `json:"text"`
			} `json:"results"`
		} `json:"payload"`
	}
	if json.Unmarshal([]byte(line[i+len("instruction="):]), &item) != nil {
		return Event{}
	}
	if item.Header.Namespace == "SpeechSynthesizer" && item.Header.Name == "Speak" {
		return Event{"speak", item.Payload.Text}
	}
	if item.Header.Namespace == "SpeechRecognizer" && item.Header.Name == "RecognizeResult" && item.Payload.Final && len(item.Payload.Results) > 0 {
		return Event{"query", item.Payload.Results[0].Text}
	}
	return Event{}
}

func hasAny(s string, values ...string) bool {
	for _, v := range values {
		if strings.Contains(s, v) {
			return true
		}
	}
	return false
}

// NativeRequest conservatively preserves the speaker's existing device actions.
// Open-ended questions default to DeepSeek, with no required opening phrase.
func NativeRequest(query string) bool {
	q := strings.TrimSpace(query)
	q = strings.TrimLeft(strings.TrimPrefix(q, "请问"), " ，,。？！?!")
	// These questions need the native assistant's live local forecast, even
	// though they do not explicitly contain the word "weather".
	if hasAny(q, "今天", "明天", "后天", "现在", "今晚", "今早", "明早", "周末", "这周", "下周", "出门", "外面") && hasAny(q, "穿什么", "怎么穿", "穿多少", "穿几件", "穿多厚", "穿衣", "短袖", "长袖", "外套", "羽绒服", "凉鞋", "带伞", "雨伞", "晒衣", "晒被", "洗车", "跑步") {
		return true
	}
	if strings.HasPrefix(q, "穿什么衣服") || strings.HasPrefix(q, "要带伞") || strings.HasPrefix(q, "需要带伞") || strings.HasPrefix(q, "外面冷吗") {
		return true
	}
	if hasAny(q, "为什么", "原理", "解释", "什么是", "介绍一下", "帮我写", "写一首", "写一篇", "翻译", "总结一下", "讲讲", "讲一个", "编一个", "设计一个") {
		return false
	}
	if hasAny(q, "天气", "空气质量", "紫外线", "日出时间", "日落时间") {
		return true
	}
	if hasAny(q, "下雨", "下雪", "气温", "温度", "几度", "带伞", "晒被子") && hasAny(q, "今天", "明天", "后天", "现在", "周末", "这周", "下周") {
		return true
	}
	if hasAny(q, "今天几号", "现在几点", "几点了", "星期几", "礼拜几", "农历", "音量", "声音大", "声音小", "大声点", "小声点", "静音", "停止播放", "暂停", "继续播放", "换一首", "上一首", "下一首") {
		return true
	}
	if hasAny(q, "闹钟", "倒计时", "计时器", "提醒我", "叫我起床", "关闭提醒", "取消提醒") {
		return true
	}
	if hasAny(q, "打开", "关闭", "开启", "关掉", "关上", "开一下", "关一下", "开大", "关小", "关了", "开灯", "关灯", "调高", "调低", "亮一点", "暗一点", "进入睡眠模式") {
		return true
	}
	if strings.HasPrefix(q, "播放") || hasAny(q, "放一首", "放首", "听一首", "来一首") {
		return true
	}
	if hasAny(q, "空调", "窗帘", "卧室灯", "客厅灯", "风扇", "加湿器", "扫地机", "电视", "插座") && hasAny(q, "设为", "设置", "调到", "状态", "开了吗", "关了吗", "开着吗", "关着吗") {
		return true
	}
	for _, v := range []string{"你好", "谢谢", "再见", "小爱同学", "停止", "停下", "不用了"} {
		if strings.TrimRight(q, "，。？！!?") == v {
			return true
		}
	}
	return false
}

func SpeechText(s string) string {
	s = strings.NewReplacer("**", "", "#", "", "`", "", "\n", " ").Replace(strings.TrimSpace(s))
	r := []rune(s)
	if len(r) > 240 {
		s = string(r[:239]) + "。"
	}
	return s
}
