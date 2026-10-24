// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"xiaobot"
	"xiaobot/deepseekweb"
	"xiaobot/jarvis/openai"
	"xiaobot/runtimeenv"
	"xiaobot/voiceanswer"
)

func main() {
	path := flag.String("config", "/data/adb/lx04-deepseek/config.json", "private config")
	query := flag.String("query", "联网搜索现在排名高的大模型", "bounded live-search test")
	flag.Parse()
	xiaobot.FixedZone()
	if runtimeenv.ConfigureDNSFromEnv() != nil {
		fmt.Println("DNS setup failed")
		os.Exit(1)
	}
	cfg, err := xiaobot.NewConfigFromFile(*path)
	if err != nil {
		fmt.Println("Private config unavailable")
		os.Exit(1)
	}
	client := openai.NewClient(cfg.OpenAIKey, cfg.OpenAIBackend, "")
	client.GPTOptions = cfg.GPTOptions
	e := voiceanswer.Engine{Chat: client, Search: deepseekweb.NewClient(cfg.OpenAIKey), Model: cfg.Bot, Prompt: cfg.Prompt}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	answer, err := e.Answer(ctx, *query)
	if err != nil {
		fmt.Println("Live-search test failed; authentication and response details suppressed")
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(answer, "", "  ")
	os.WriteFile(filepath.Join(filepath.Dir(*path), "search-check.json"), data, 0600)
	fmt.Println("Search calls:", len(answer.Searches))
	for _, s := range answer.Searches {
		for _, source := range s.Sources {
			fmt.Println("Source:", source.Title, source.URL)
		}
	}
	fmt.Println("Answer:", answer.Text)
}
