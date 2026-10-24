package webui

import (
	"encoding/json"
	"net/http"
	"ninego/log"
)

type Chat struct {
	Message    string `json:"message"`
	PlayQuery  bool   `json:"playQuery"`
	PlayAnswer bool   `json:"playAnswer"`
	Stream     bool   `json:"stream"`
}

func do_Chat(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || request.Body == nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	if bot != nil {
		var chat Chat
		/*body, _ := io.ReadAll(request.Body)
		fmt.Println("Received body:", string(body))
		request.Body = io.NopCloser(bytes.NewBuffer(body)) // 注意：读取后需重置 r.Body 才能继续解析		*/
		err := json.NewDecoder(request.Body).Decode(&chat)
		if err != nil {
			log.Error("Error:", err)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		log.Debug("body->chat", chat)
		chatMode := 0
		if chat.PlayQuery && chat.PlayAnswer {
			chatMode = 3
		} else if chat.PlayQuery {
			chatMode = 1
		} else if chat.PlayAnswer {
			chatMode = 2
		}
		log.Println("chatmode=", chatMode)
		if (chatMode == 0 || chat.Stream) && config.Stream && bot.HasGPT {
			// 流式响应处理
			writer.Header().Set("Content-Type", "text/event-stream")
			writer.Header().Set("Cache-Control", "no-cache")
			writer.Header().Set("Connection", "keep-alive")
			writer.Header().Set("Access-Control-Allow-Origin", "*") // 跨域支持
			err := bot.ChatStream(chat.Message, writer)
			if err != nil {
				//http.Error 在ChatStream已调用过了
				log.Error("stream chat error:", err)
			}
		} else {
			// 普通响应处理
			bot.Talk.StreamWriter = writer
			answer := bot.Chat(chat.Message, chatMode)
			resp := struct {
				//Code    int    `json:"code"`
				Message string `json:"message"`
			}{
				//Code:    100,
				Message: answer,
			}
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusOK)
			json.NewEncoder(writer).Encode(resp)
		}
	} else {
		//writer.WriteHeader(http.StatusOK)
		writer.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(writer).Encode(map[string]string{"error": "bot is not initialized"})
	}
}
