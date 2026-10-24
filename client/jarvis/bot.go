package jarvis

import "xiaobot/jarvis/openai"

type Jarvis interface {
	Ask(msg, answer string) (string, error)
	AskStream(msg, answer string) (Stream *openai.ChatCompletionStream, err error)

	GetHistory() *[]RoleContent
	SetPrompt(newPrompt string)
}
