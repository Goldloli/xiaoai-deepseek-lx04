package jarvis

import (
	"context"
	"fmt"

	//"crypto/tls"
	//"net/http"
	"ninego/log"
	"xiaobot/jarvis/openai"
	"xiaobot/jsengine"
)

// ChatGPTResponseBody 请求体
type ChatGPTResponseBody struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"`
	Created int                      `json:"created"`
	Model   string                   `json:"model"`
	Choices []map[string]interface{} `json:"choices"`
	Usage   map[string]interface{}   `json:"usage"`
}

type ChoiceItem struct {
	Index        int         `json:"index"`
	Message      RoleContent `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type RoleContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatGPTRequestBody 响应体
type ChatGPTRequestBody struct {
	Model            string        `json:"model"`
	Prompt           string        `json:"prompt,omitempty"`
	MaxTokens        int           `json:"max_tokens"`
	Temperature      float32       `json:"temperature"`
	TopP             int           `json:"top_p"`
	FrequencyPenalty int           `json:"frequency_penalty"`
	PresencePenalty  float32       `json:"presence_penalty"`
	Stop             []string      `json:"stop"`
	Messages         []RoleContent `json:"messages,omitempty"`
}

type GhatGPT struct {
	Model   string
	Key     string
	Backend string
	Proxy   string

	Prompt         string
	GPTOptions     map[string]interface{}
	Adapter        *jsengine.Program
	HistoryMessage []RoleContent
}

func (g *GhatGPT) Ask(msg, answer string) (string, error) {

	client := openai.NewClient(g.Key, g.Backend, g.Proxy)
	client.GPTOptions = g.GPTOptions
	client.Adapter = g.Adapter

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: g.Prompt, // "system系统提示词"
		},
	}
	if g.HistoryMessage != nil {
		for i := 0; i < len(g.HistoryMessage); i++ {
			messages = append(messages,
				openai.ChatCompletionMessage{
					Role:    g.HistoryMessage[i].Role,
					Content: g.HistoryMessage[i].Content,
				},
			)
		}
	}
	if answer == "" {
		messages = append(messages,
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: msg,
			},
		)
	} else {
		messages = append(messages,
			openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleUser,
				Content: fmt.Sprintf(`你需要处理用户提问和AI回答的内容，判断AI回答是否有表述其为不知道，如果是则重新思考后给出答复，否则直接返回AI回答的原文内容。
以下是用户的提问：
<msg>
%s
</msg>
以下是AI的回答：
<answer>
%s
</answer>
不知道的表述包含以下关键词：-
- 我暂时还回答不上
- 被你问住了
- 把我难住了
- 被难住了
- 我还在学习中
- 我还在研究中
- 要再学习
- 要更努力学习
- 换个话题
- 换个方式再说一遍
- 最新小爱音箱APP
- 本设备暂不支持该功能
如果AI的回答中包含上述任意关键词，请重新给出答复（直接输出答案，不要带有任何其他信息，不要让人知道有AI的存在）；如果不包含，则直接返回AI回答的原文内容。`, msg, answer),
			},
		)
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: g.Model, //openai.GPT3Dot5Turbo,
			/*Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: g.Prompt, // "system系统提示词"
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: msg,
				},
			},*/
			Messages:         messages,
			Temperature:      0.8,
			TopP:             1,
			Stop:             []string{},
			PresencePenalty:  0.6,
			FrequencyPenalty: 0,
		},
	)
	log.Debug("gpt resp:", resp)
	if err != nil {
		log.Error("ChatCompletion error: ", err)
		return "", err
	}

	//return resp.Choices[0].Message.Content, nil
	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	} else {
		return "", nil
	}
}

func (g *GhatGPT) AskStream(msg, answer string) (Stream *openai.ChatCompletionStream, err error) {

	client := openai.NewClient(g.Key, g.Backend, g.Proxy)
	client.GPTOptions = g.GPTOptions
	client.Adapter = g.Adapter

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: g.Prompt, // "system系统提示词"
		},
	}
	if g.HistoryMessage != nil {
		for i := 0; i < len(g.HistoryMessage); i++ {
			messages = append(messages,
				openai.ChatCompletionMessage{
					Role:    g.HistoryMessage[i].Role,
					Content: g.HistoryMessage[i].Content,
				},
			)
		}
	}
	if answer == "" {
		messages = append(messages,
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: msg,
			},
		)
	} else {
		messages = append(messages,
			openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleUser,
				Content: fmt.Sprintf(`你需要处理用户提问和AI回答的内容，判断AI回答是否有表述其为不知道，如果是则重新思考后给出答复，否则直接返回AI回答的原文内容。
以下是用户的提问：
<msg>
%s
</msg>
以下是AI的回答：
<answer>
%s
</answer>
不知道的表述包含以下关键词：-
- 我暂时还回答不上
- 被你问住了
- 把我难住了
- 被难住了
- 我还在学习中
- 我还在研究中
- 要再学习
- 要更努力学习
- 换个话题
- 换个方式再说一遍
- 最新小爱音箱APP
- 本设备暂不支持该功能
如果AI的回答中包含上述任意关键词，请重新给出答复（直接输出答案，不要带有任何其他信息，不要让人知道有AI的存在）；如果不包含，则直接返回AI回答的原文内容。`, msg, answer),
			},
		)
	}

	stream, err := client.CreateChatCompletionStream(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: g.Model, //openai.GPT3Dot5Turbo,
			/*Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: g.Prompt, // "system系统提示词"
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: msg,
				},
			},*/
			Messages:         messages,
			Temperature:      0.8,
			TopP:             1,
			Stop:             []string{},
			PresencePenalty:  0.6,
			FrequencyPenalty: 0,
		},
	)

	if err != nil {
		log.Error("ChatCompletionStream error: ", err)
		return nil, err
	}

	return stream, nil
}

func (g *GhatGPT) GetHistory() *[]RoleContent {
	return &g.HistoryMessage
}

func (g *GhatGPT) SetPrompt(newPrompt string) {
	g.Prompt = newPrompt + "\n以下请只回答文字不要带链接，回答内容尽量精明简短，不要超过100字"
}
