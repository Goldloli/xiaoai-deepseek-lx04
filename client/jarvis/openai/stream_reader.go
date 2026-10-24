package openai

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"xiaobot/jsengine"
)

var (
	headerData  = regexp.MustCompile(`^data:\s*`)
	errorPrefix = regexp.MustCompile(`^data:\s*{"error":`)

	ErrTooManyEmptyStreamMessages = errors.New("stream has sent too many empty messages")
)

type streamReader struct {
	emptyMessagesLimit uint
	isFinished         bool

	reader         *bufio.Reader
	response       *http.Response
	adapter        *jsengine.Program
	errAccumulator ErrorAccumulator
	unmarshaler    Unmarshaler

	httpHeader http.Header
}

/*openai: 流式响应格式
  	data: {"choices":[{"delta":{"content":"你好"}}]}\n\n
  	data: {"choices":[{"delta":{"content":"世界"}}]}\n\n
  	data: [DONE]\n\n
  Ollama 流式响应格式
  	{"response":"你好", "done":false}\n
  	{"response":"世界", "done":false}\n
  	{"response":"", "done":true}\n
  AnythingLLM 流式结构
	data: {
	  "id": "stream-xxx",
	  "type": "chunk",
	  "error": null,	   <--
	  "content": "Hello",  // 直接在顶层返回内容，而非嵌套在 choices.delta 中
	  "done": false
	}
  openai流式返回为data:加json或是[DONE]; 而ollama流式返回的是一个json
*/

func (stream *streamReader) Recv() (response ChatCompletionStreamResponse, err error) {
	rawLine, err := stream.RecvRaw()
	if err != nil {
		return
	}

	err = stream.unmarshaler.Unmarshal(rawLine, &response)
	if err != nil {
		return
	}
	return response, nil
}

func (stream *streamReader) RecvRaw() ([]byte, error) {
	if stream.isFinished {
		return nil, io.EOF
	}

	return stream.processLines()
}

//nolint:gocognit
func (stream *streamReader) processLines() ([]byte, error) {
	/*var (
		emptyMessagesCount uint
		hasErrorPrefix     bool
	)
	for {
		rawLine, readErr := stream.reader.ReadBytes('\n')
		fmt.Println("Line=>", string(rawLine))
		if (readErr != nil) || hasErrorPrefix {
			respErr := stream.unmarshalError()
			if respErr != nil {
				return nil, fmt.Errorf("error, %w", respErr.Error)
			}
			return nil, readErr
		}

		noSpaceLine := bytes.TrimSpace(rawLine)
		if errorPrefix.Match(noSpaceLine) {
			hasErrorPrefix = true
		}
		if !headerData.Match(noSpaceLine) || hasErrorPrefix {
			if hasErrorPrefix {
				noSpaceLine = headerData.ReplaceAll(noSpaceLine, nil)
			}
			writeErr := stream.errAccumulator.Write(noSpaceLine)
			if writeErr != nil {
				return nil, writeErr
			}
			emptyMessagesCount++
			if emptyMessagesCount > stream.emptyMessagesLimit {
				return nil, ErrTooManyEmptyStreamMessages
			}
			continue
		}
		noPrefixLine := headerData.ReplaceAll(noSpaceLine, nil)
		// 适配器转换
		if stream.adapter != nil {
			jsonBytes, err := stream.adapter.ConvertResponse(noPrefixLine, true)
			if err != nil {
				return nil, fmt.Errorf("adapter convert response failed: %w", err)
			}
			noPrefixLine = jsonBytes
		}
		if string(noPrefixLine) == "[DONE]" {
			stream.isFinished = true
			return nil, io.EOF
		}

		return noPrefixLine, nil
	}*/

	var (
		//emptyMessagesCount uint
		hasErrorPrefix bool
	)
	for {
		rawLine, readErr := stream.reader.ReadBytes('\n')
		//fmt.Println("Line=>", readErr, string(rawLine))
		if (readErr != nil && !errors.Is(readErr, io.EOF)) || hasErrorPrefix {
			respErr := stream.unmarshalError()
			if respErr != nil {
				return nil, fmt.Errorf("error, %w", respErr.Error)
			}
			return nil, readErr
		}

		noSpaceLine := bytes.TrimSpace(rawLine)
		if headerData.Match(noSpaceLine) {
			noPrefixLine := headerData.ReplaceAll(noSpaceLine, nil)
			// 适配器转换
			if stream.adapter != nil {
				jsonBytes, err := stream.adapter.ConvertResponse(noPrefixLine, true)
				if err != nil {
					return nil, fmt.Errorf("adapter convert response failed: %w", err)
				}
				noPrefixLine = jsonBytes
			}
			//fmt.Println("noPrefixLine->", string(noPrefixLine))
			if string(noPrefixLine) == "[DONE]" {
				stream.isFinished = true
				return nil, io.EOF
			}
			return noPrefixLine, nil
		} else {
			// 适配器转换
			if stream.adapter != nil && string(noSpaceLine) != "" {
				jsonBytes, err := stream.adapter.ConvertResponse(noSpaceLine, true)
				if err != nil {
					return nil, fmt.Errorf("adapter convert response failed: %w", err)
				}
				noSpaceLine = jsonBytes
			}
			//fmt.Println("noSpaceLine->", string(noSpaceLine))
			if stream.isFinished {
				return nil, io.EOF
			}
			if errors.Is(readErr, io.EOF) {
				stream.isFinished = true
				if string(noSpaceLine) == "" {
					return nil, io.EOF
				}
				return noSpaceLine, nil
			}
			if string(noSpaceLine) == "" {
				continue
			}
			return noSpaceLine, nil
		}

	}
}

func (stream *streamReader) unmarshalError() (errResp *ErrorResponse) {
	errBytes := stream.errAccumulator.Bytes()
	if len(errBytes) == 0 {
		return
	}

	err := stream.unmarshaler.Unmarshal(errBytes, &errResp)
	if err != nil {
		errResp = nil
	}

	return
}

func (stream *streamReader) Close() error {
	return stream.response.Body.Close()
}
