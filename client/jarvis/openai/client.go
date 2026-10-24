// Modified or added for XiaoSeek LX04, 2026-10-24. See NOTICE.
// SPDX-License-Identifier: Apache-2.0
package openai

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"ninego/log"
	"strings"
	"time"
	"xiaobot/jsengine"
)

type Client struct {
	apiKey  string
	baseURL string
	client  *http.Client

	GPTOptions map[string]interface{}
	Adapter    *jsengine.Program
}

func NewClient(apiKey string, baseURL string, proxyURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1/chat/completions"
	} else if strings.HasSuffix(baseURL, "/") {
		baseURL = strings.TrimSuffix(baseURL, "/")
	} else if !strings.HasSuffix(baseURL, "/chat/completions") {
		baseURL = fmt.Sprintf("%s/chat/completions", baseURL)
	}

	transport := &http.Transport{
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 90 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxy)
		}
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Transport: transport, Timeout: 120 * time.Second},
	}
}

// NewClientWithConfig creates new OpenAI API client for specified config.
/*func NewClientWithConfig(config ClientConfig) *Client {
	return &Client{
		config:         config,
		requestBuilder: utils.NewRequestBuilder(),
		createFormBuilder: func(body io.Writer) utils.FormBuilder {
			return utils.NewFormBuilder(body)
		},
	}
}*/

/*
	func (c *Client) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
		body, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request: %w", err)
		}

		if c.GPTOptions != nil && len(c.GPTOptions) > 0 {
			var m map[string]interface{}
			json.Unmarshal(body, &m)

			data, _ := json.Marshal(c.GPTOptions)
			json.Unmarshal(data, &m)

			body, _ = json.Marshal(m)
		}

		httpReq, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			c.baseURL,
			bytes.NewReader(body),
		)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("error making request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errResp struct {
				Error struct {
					Message string `json:"message"`
					Type    string `json:"type"`
					Code    string `json:"code"`
				} `json:"error"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
				return nil, fmt.Errorf("error response with status %d", resp.StatusCode)
			}
			return nil, fmt.Errorf("%s: %s", errResp.Error.Type, errResp.Error.Message)
		}

		var response ChatCompletionResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("error decoding response: %w", err)
		}

		return &response, nil
	}
*/
func (c *Client) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// 序列化请求体
	body, err := json.Marshal(req)
	log.Debug("body-->", string(body))
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// 处理适配器转换
	header := make(map[string]string)
	if c.Adapter != nil {
		jsonBytes, err := c.Adapter.ConvertRequest(header, body)
		if err != nil {
			return nil, fmt.Errorf("adapter convert request failed: %w", err)
		}
		body = jsonBytes
	}

	// 合并GPT选项
	if c.GPTOptions != nil && len(c.GPTOptions) > 0 {
		var reqMap map[string]interface{}
		// 解析现有请求体到map
		if err := json.Unmarshal(body, &reqMap); err != nil {
			return nil, fmt.Errorf("error unmarshaling request body: %w", err)
		}

		// 解析GPTOptions到临时map
		optionsData, err := json.Marshal(c.GPTOptions)
		if err != nil {
			return nil, fmt.Errorf("error marshaling gpt options: %w", err)
		}

		var optionsMap map[string]interface{}
		if err := json.Unmarshal(optionsData, &optionsMap); err != nil {
			return nil, fmt.Errorf("error unmarshaling gpt options: %w", err)
		}

		// 合并选项到请求map
		for k, v := range optionsMap {
			reqMap[k] = v
		}

		// 重新序列化合并后的请求
		mergedBody, err := json.Marshal(reqMap)
		if err != nil {
			return nil, fmt.Errorf("error marshaling merged request: %w", err)
		}
		body = mergedBody
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	for k, v := range header {
		httpReq.Header.Set(k, v)
	}

	log.Debug("convertBody-->", string(body))
	// 发送请求
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close() // 确保响应体被关闭

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}

		// 解码错误响应
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(&errResp); err != nil {
			return nil, fmt.Errorf("non-200 status code: %d, error decoding response: %w",
				resp.StatusCode, err)
		}

		return nil, fmt.Errorf("%s: %s (code: %s)",
			errResp.Error.Type, errResp.Error.Message, errResp.Error.Code)
	}

	// 处理响应
	var response ChatCompletionResponse
	if c.Adapter != nil {
		// 读取响应体
		respBody, err := ioutil.ReadAll(resp.Body)
		log.Debug("respBody-->", string(respBody))
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		// 适配器转换
		jsonBytes, err := c.Adapter.ConvertResponse(respBody, false)
		log.Debug("convertResp-->", string(jsonBytes))
		if err != nil {
			return nil, fmt.Errorf("adapter convert response failed: %w", err)
		}
		if err := json.Unmarshal(jsonBytes, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal to ChatCompletionResponse: %w", err)
		}
	} else {
		// 直接解码到目标结构体
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(&response); err != nil {
			return nil, fmt.Errorf("error decoding response: %w", err)
		}
	}

	return &response, nil
}

func (c *Client) CreateChatCompletionStream(ctx context.Context, request ChatCompletionRequest) (stream *ChatCompletionStream, err error) {
	request.Stream = true

	body, err := json.Marshal(request)
	log.Debug("streamBody--", string(body))
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// 处理适配器转换
	header := make(map[string]string)
	if c.Adapter != nil {
		jsonBytes, err := c.Adapter.ConvertRequest(header, body)
		if err != nil {
			return nil, fmt.Errorf("adapter convert request failed: %w", err)
		}
		body = jsonBytes
		log.Debug("convertBody--", string(body))
	}
	if c.GPTOptions != nil && len(c.GPTOptions) > 0 {
		var m map[string]interface{}
		json.Unmarshal(body, &m)

		data, _ := json.Marshal(c.GPTOptions)
		json.Unmarshal(data, &m)

		body, _ = json.Marshal(m)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	for k, v := range header {
		req.Header.Set(k, v)
	}

	/*resp, err := c.client.Do(req) //nolint:bodyclose // body is closed in stream.Close()
	if err != nil {
		return new(streamReader[T]), err
	}
	if resp.StatusCode != http.StatusOK {
		if isFailureStatusCode(resp) {
			return new(streamReader[T]), client.handleErrorResp(resp)
		}
	}

	response := &streamReader[T]{
		emptyMessagesLimit: client.config.EmptyMessagesLimit,
		reader:             bufio.NewReader(resp.Body),
		response:           resp,
		errAccumulator:     NewErrorAccumulator(),
		unmarshaler:        &JSONUnmarshaler{},
		httpHeader:         httpHeader(resp.Header),
	}
	stream = &ChatCompletionStream{
		streamReader: resp,
	}
	*/

	resp, err := sendRequestStream(c, req)
	if err != nil {
		log.Debug("sendRequestStream Error", err)
		return
	}
	stream = &ChatCompletionStream{
		streamReader: resp,
	}

	return
}

func sendRequestStream(c *Client, req *http.Request) (*streamReader, error) {
	/*req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")*/

	resp, err := c.client.Do(req) //nolint:bodyclose // body is closed in stream.Close()
	if err != nil {
		return new(streamReader), err
	}
	//if isFailureStatusCode(resp) {
	//	return new(streamReader), client.handleErrorResp(resp)
	//}
	return &streamReader{
		emptyMessagesLimit: 300, //client.config.EmptyMessagesLimit,
		reader:             bufio.NewReader(resp.Body),
		response:           resp,
		errAccumulator:     NewErrorAccumulator(),
		unmarshaler:        &JSONUnmarshaler{},
		httpHeader:         http.Header(resp.Header),
		adapter:            c.Adapter,
	}, nil
}

//ErrorAccumulator

type ErrorAccumulator interface {
	Write(p []byte) error
	Bytes() []byte
}

type errorBuffer interface {
	io.Writer
	Len() int
	Bytes() []byte
}

type DefaultErrorAccumulator struct {
	Buffer errorBuffer
}

func NewErrorAccumulator() ErrorAccumulator {
	return &DefaultErrorAccumulator{
		Buffer: &bytes.Buffer{},
	}
}

func (e *DefaultErrorAccumulator) Write(p []byte) error {
	_, err := e.Buffer.Write(p)
	if err != nil {
		return fmt.Errorf("error accumulator write error, %w", err)
	}
	return nil
}

func (e *DefaultErrorAccumulator) Bytes() (errBytes []byte) {
	if e.Buffer.Len() == 0 {
		return
	}
	errBytes = e.Buffer.Bytes()
	return
}

//Unmarshaler

type Unmarshaler interface {
	Unmarshal(data []byte, v any) error
}

type JSONUnmarshaler struct{}

func (jm *JSONUnmarshaler) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
