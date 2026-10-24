// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
package voiceanswer

import (
	"context"
	"strings"
	"testing"
	"time"
	"xiaobot/deepseekweb"
	"xiaobot/jarvis/openai"
)

type chatFunc func(context.Context, openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error)

func (f chatFunc) CreateChatCompletion(c context.Context, r openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	return f(c, r)
}

type searchFunc func(context.Context, string, string) (deepseekweb.Result, error)

func (f searchFunc) Search(c context.Context, q, o string) (deepseekweb.Result, error) {
	return f(c, q, o)
}
func TestCurrentRankingHasRealSourcesBeforeAnswer(t *testing.T) {
	searched := 0
	e := Engine{Model: "test", Now: func() time.Time { return time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC) }}
	e.Search = searchFunc(func(_ context.Context, q, o string) (deepseekweb.Result, error) {
		searched++
		if !strings.Contains(q, "2026-10") {
			t.Fatal("missing current date")
		}
		return deepseekweb.Result{Results: []deepseekweb.Source{{Title: "Benchmark", URL: "https://example.com", Summary: "verified ranking"}}}, nil
	})
	e.Chat = chatFunc(func(_ context.Context, r openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
		if searched != 1 || len(r.Messages) != 4 || r.Messages[3].Role != "tool" || !strings.Contains(r.Messages[3].Content, "verified ranking") {
			t.Fatal("no verified tool evidence")
		}
		return &openai.ChatCompletionResponse{Choices: []openai.Choice{{Message: openai.ChatCompletionMessage{Content: "根据榜单回答"}}}}, nil
	})
	a, err := e.Answer(context.Background(), "现在排名高的大模型")
	if err != nil || len(a.Searches) != 1 || a.Text == "" {
		t.Fatal(a, err)
	}
}
func TestFailedSearchDoesNotInventCurrentAnswer(t *testing.T) {
	e := Engine{Search: searchFunc(func(context.Context, string, string) (deepseekweb.Result, error) {
		return deepseekweb.Result{}, deepseekweb.ErrSearch
	}), Chat: chatFunc(func(context.Context, openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
		t.Fatal("model used without required fresh evidence")
		return nil, nil
	})}
	if _, err := e.Answer(context.Background(), "搜索最新模型排行榜"); err == nil {
		t.Fatal("failed search accepted")
	}
}

func TestModelCanChooseSearchWithoutFixedKeyword(t *testing.T) {
	round := 0
	searched := 0
	e := Engine{Search: searchFunc(func(context.Context, string, string) (deepseekweb.Result, error) {
		searched++
		return deepseekweb.Result{Results: []deepseekweb.Source{{Title: "Official", Summary: "evidence"}}}, nil
	})}
	e.Chat = chatFunc(func(_ context.Context, r openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
		round++
		message := openai.ChatCompletionMessage{Content: "据资料回答"}
		if round == 1 {
			message.Content = ""
			message.ToolCalls = []openai.ToolCall{{ID: "call_test", Type: "function", Function: openai.FunctionCall{Name: "web_search", Arguments: `{"query":"a current product"}`}}}
		} else if r.Messages[len(r.Messages)-1].ToolCallID != "call_test" {
			t.Fatal("tool result lost its call id")
		}
		return &openai.ChatCompletionResponse{Choices: []openai.Choice{{Message: message}}}, nil
	})
	a, err := e.Answer(context.Background(), "这个新产品有什么特点")
	if err != nil || searched != 1 || round != 2 || a.Text == "" {
		t.Fatal(a, err)
	}
}

func TestStableQuestionCanSkipSearch(t *testing.T) {
	e := Engine{Search: searchFunc(func(context.Context, string, string) (deepseekweb.Result, error) {
		t.Fatal("unnecessary search")
		return deepseekweb.Result{}, nil
	}), Chat: chatFunc(func(context.Context, openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
		return &openai.ChatCompletionResponse{Choices: []openai.Choice{{Message: openai.ChatCompletionMessage{Content: "二"}}}}, nil
	})}
	if a, err := e.Answer(context.Background(), "一加一等于几"); err != nil || len(a.Searches) != 0 {
		t.Fatal(a, err)
	}
}
