// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
package xiaobot

import "testing"

func TestNativeCommandsNeverTouchBotOrSpeaker(t *testing.T) {
	for _, query := range []string{"今天天气怎么样", "打开卧室灯", "把音量调到三十", "播放音乐", "你是谁"} {
		// No Box or Bot is supplied: reaching any interception path would panic.
		talk := NewMiTalk(&Config{Keywords: []string{"请问"}})
		if err := talk.handleConversation(Record{Query: query}, false); err != nil {
			t.Fatal(err)
		}
	}
}
