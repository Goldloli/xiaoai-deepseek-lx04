package webui

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"ninego/log"
	"xiaobot/jsengine"
)

//在同一次请求响应过程中，只能调用一次WriteHeader(code int)，否则会有一条日志输出“http: superfluous response.WriteHeader call from”。
type ResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *ResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
	w.wroteHeader = true
}

func do_Task(writer http.ResponseWriter, request *http.Request) {
	act := request.PathValue("action")
	jsScript := config.TaskJS[act]
	log.Println("execute action:", act)
	rw := &ResponseWriter{ResponseWriter: writer}
	if err := jsengine.Do_Action(rw, request, &jsScript); err == nil {
		if !rw.wroteHeader {
			writer.WriteHeader(http.StatusOK)
		}
	}
}

func do_queryTest(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || request.Body == nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	req := struct {
		Question      string `json:"question"`
		ScriptContent string `json:"scriptContent"`
	}{}
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		log.Error("request Error:", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := jsengine.Exec_queryJS(req.Question, req.ScriptContent); err != nil {
		log.Error("Execute query JavaScript Error:", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("test 'query.bot' - ok!")
	// 设置正确的Content-Type
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte("success"))
}
func do_querySave(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || request.Body == nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	// 更安全的读取方式
	body, err := io.ReadAll(request.Body)
	if err != nil {
		log.Error("Read body failed:", err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	defer request.Body.Close() // 确保关闭body
	err = ioutil.WriteFile("query.bot", body, 0666)
	if err != nil {
		log.Error("Write file 'query.bot' failed:", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("Write file 'query.bot' - ok")
	config.QueryJS = string(body)
	// 设置正确的Content-Type
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
}
