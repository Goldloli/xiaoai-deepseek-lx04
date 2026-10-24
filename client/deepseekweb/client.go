// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
// Package deepseekweb uses DeepSeek's authenticated, read-only Search API.
package deepseekweb

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Source struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
	Summary string `json:"summary,omitempty"`
	Date    string `json:"date,omitempty"`
}
type Result struct {
	Results []Source `json:"results"`
}
type Client struct {
	key, endpoint string
	http          *http.Client
}

var ErrSearch = errors.New("web search could not be verified")

func NewClient(key string) *Client {
	return &Client{key: key, endpoint: "https://api.deepseek.com/v1/tools/search", http: &http.Client{Timeout: 9 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 5 * time.Second, IdleConnTimeout: 30 * time.Second}}}
}
func bounded(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > n {
		return string(r[:n])
	}
	return string(r)
}
func (c *Client) Search(ctx context.Context, query, original string) (Result, error) {
	var result Result
	query = bounded(query, 128)
	if query == "" {
		return result, ErrSearch
	}
	body, _ := json.Marshal(map[string]any{"query": query, "ori_question": bounded(original, 128), "count": 5, "need_content": false})
	ctx, cancel := context.WithTimeout(ctx, 9*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return result, ErrSearch
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return result, fmt.Errorf("%w (HTTP %d)", ErrSearch, resp.StatusCode)
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&result) != nil || len(result.Results) == 0 {
		return Result{}, ErrSearch
	}
	if len(result.Results) > 5 {
		result.Results = result.Results[:5]
	}
	for i := range result.Results {
		result.Results[i].Summary = bounded(result.Results[i].Summary, 1200)
		result.Results[i].Snippet = bounded(result.Results[i].Snippet, 400)
	}
	return result, nil
}
