// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
// A bounded connectivity test; credentials are read locally and never printed.
package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"xiaobot/runtimeenv"
)

func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }

func main() {
	config := flag.String("config", "", "private config file")
	chat := flag.Bool("chat", false, "send a small connection test")
	flag.Parse()
	if err := runtimeenv.ConfigureDNSFromEnv(); err != nil {
		fail(err.Error())
	}
	data, err := os.ReadFile(*config)
	if err != nil {
		fail("Cannot read private config")
	}
	var private struct {
		Key string `json:"openai_key"`
	}
	if json.Unmarshal(data, &private) != nil || private.Key == "" {
		fail("Missing API key")
	}
	client := &http.Client{Timeout: 45 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 10 * time.Second}}
	request, _ := http.NewRequest("GET", "https://api.deepseek.com/models", nil)
	request.Header.Set("Authorization", "Bearer "+private.Key)
	response, err := client.Do(request)
	if err != nil {
		fail("HTTPS connection failed: " + err.Error())
	}
	defer response.Body.Close()
	fmt.Println("DeepSeek models HTTP status:", response.StatusCode)
	if response.StatusCode != 200 {
		fail("API authentication or service check failed; response body suppressed")
	}
	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&models) != nil {
		fail("Invalid models response")
	}
	available := map[string]bool{}
	for _, model := range models.Data {
		available[model.ID] = true
		fmt.Println("Available model:", model.ID)
	}
	selected := ""
	for _, name := range []string{"deepseek-flash", "deepseek-v4-flash", "deepseek-chat", "deepseek-v4-pro"} {
		if available[name] {
			selected = name
			break
		}
	}
	if selected == "" {
		fail("No recognized conversational model; inspect available model IDs")
	}
	fmt.Println("Selected model:", selected)
	if !*chat {
		return
	}
	body, _ := json.Marshal(map[string]any{"model": selected, "messages": []map[string]string{{"role": "user", "content": "只回复：连接成功"}}, "max_tokens": 64, "stream": false, "thinking": map[string]string{"type": "disabled"}})
	request, _ = http.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+private.Key)
	request.Header.Set("Content-Type", "application/json")
	answer, err := client.Do(request)
	if err != nil {
		fail("Chat connection failed: " + err.Error())
	}
	defer answer.Body.Close()
	fmt.Println("DeepSeek chat HTTP status:", answer.StatusCode)
	if answer.StatusCode != 200 {
		fail("Chat service check failed; response body suppressed")
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.NewDecoder(io.LimitReader(answer.Body, 1<<20)).Decode(&result) != nil || len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		fail("Chat returned no answer")
	}
	fmt.Println("Chat answer:", result.Choices[0].Message.Content)
}
