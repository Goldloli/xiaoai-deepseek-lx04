// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
package devicevoice

import "testing"

func TestAutomaticRouting(t *testing.T) {
	for _, q := range []string{"今天天气怎么样", "请问明天天气怎么样", "明天穿什么衣服", "今天需要带伞吗", "周末适合晒被子吗", "明天可以穿短袖吗", "要带伞吗", "把卧室灯打开", "帮我关闭空调", "空调开着吗", "播放周杰伦的晴天", "设置明天七点的闹钟", "音量小一点", "现在几点", "停止播放"} {
		if !NativeRequest(q) {
			t.Errorf("native command intercepted: %s", q)
		}
	}
	for _, q := range []string{"彩虹是怎么形成的", "唐诗三百首中最好的一首诗是什么", "为什么会下雨", "古代唐朝人穿什么衣服", "解释一下空调的工作原理", "什么是闹钟的机械原理", "帮我写一首关于春天的诗", "翻译你好为英语", "给我推荐几本适合儿童的书", "请问一加一等于几"} {
		if NativeRequest(q) {
			t.Errorf("question routed to native: %s", q)
		}
	}
}
func TestLocalSpeechEvents(t *testing.T) {
	line := `instruction={"header":{"namespace":"SpeechRecognizer","name":"RecognizeResult"},"payload":{"is_final":true,"results":[{"text":"彩虹怎么形成"}]}}`
	if e := ParseEvent(line); e.Kind != "query" || e.Text != "彩虹怎么形成" {
		t.Fatal(e)
	}
	if e := ParseEvent(`[SpeechManager]:MSG_ASR_FINAL_RESULT:打开卧室灯`); e.Kind != "query" || e.Text != "打开卧室灯" {
		t.Fatal(e)
	}
	if e := ParseEvent(`instruction={"header":{"namespace":"SpeechRecognizer","name":"RecognizeResult"},"payload":{"is_final":false,"results":[{"text":"彩虹"}]}}`); e.Kind != "" {
		t.Fatal("partial ASR triggered a request")
	}
	if e := ParseEvent(`instruction={"header":{"namespace":"SpeechSynthesizer","name":"Speak"},"payload":{"text":"回答内容"}}`); e.Kind != "speak" || e.Text != "回答内容" {
		t.Fatal(e)
	}
}
