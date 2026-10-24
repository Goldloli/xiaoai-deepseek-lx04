package webui

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"ninego/log"
	"sort"
	"strings"
	"sync/atomic"
	"xiaobot/jsengine"
	"xiaobot/music"
)

//go:embed website/*
var Assets embed.FS

var MusicFS *DynamicMusicServer

func do_Monitoring(writer http.ResponseWriter, request *http.Request) {
	if bot != nil && ticker != nil {
		ticker.Reset(monitoringTriggerInterval)
		bot.UpdateMonitor(+1)
	}
	writer.WriteHeader(http.StatusOK)
}

func Router() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/task/{action}", do_Task)
	mux.HandleFunc("/chat", do_Chat)
	mux.HandleFunc("/monitor", do_Monitoring)
	mux.HandleFunc("/submit-config", do_setConfig)
	mux.HandleFunc("/get-config", do_getConfig)
	mux.HandleFunc("/get-devices", do_getDeviceList)

	// 创建一个子文件系统，只包含static目录下的内容
	staticFS, _ := fs.Sub(Assets, "website")
	// 将静态文件系统注册到HTTP服务器
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// 添加一个处理器，将/config路径映射到static/config.html
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		// 从嵌入的文件系统读取static/config.html
		content, _ := Assets.ReadFile("website/config.html")
		// 设置正确的Content-Type
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	})
	//提问内容js编辑
	mux.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		// 从嵌入的文件系统读取static/query_jseditor.html
		content, _ := Assets.ReadFile("website/query_jseditor.html")
		// 设置正确的Content-Type
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	})
	mux.HandleFunc("/query/script", func(w http.ResponseWriter, r *http.Request) {
		// 设置正确的Content-Type
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(config.QueryJS))
	})
	mux.HandleFunc("/query/test", do_queryTest)
	mux.HandleFunc("/query/save", do_querySave)

	//任务内容js编辑
	mux.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		// 从嵌入的文件系统读取html
		content, _ := Assets.ReadFile("website/tasks_jseditor.html")
		// 设置正确的Content-Type
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	})
	mux.HandleFunc("/task/list", func(w http.ResponseWriter, r *http.Request) {
		tasks := make([]struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Memo string `json:"memo"`
		}, len(config.TaskJS))
		// 排序
		var keys []string
		for k := range config.TaskJS {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		// 生成Slice
		for i := 0; i < len(keys); i++ {
			jsname := keys[i]
			tasks[i].ID = jsname
			tasks[i].Name = jsname
			tasks[i].Memo = strings.Split(config.TaskJS[jsname], "\n")[0]
		}
		// 设置正确的Content-Type
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(tasks)
	})
	mux.HandleFunc("/task/script/{action}", func(w http.ResponseWriter, r *http.Request) {
		taskName := r.PathValue("action")
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(config.TaskJS[taskName]))
			return
		}
		if r.Method != http.MethodPost || r.Body == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// 更安全的读取方式
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error("Read body failed:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close() // 确保关闭body
		err = ioutil.WriteFile(taskName+".bot", body, 0666)
		if err != nil {
			log.Error("Write file failed:", taskName, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Println("Write file", taskName, "- ok")
		config.TaskJS[taskName] = string(body)
		// 设置正确的Content-Type
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)

	})
	mux.HandleFunc("/task/test/{action}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Body == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 更安全的读取方式
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error("Read body failed:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close() // 确保关闭body
		jscode := string(body)
		rw := &ResponseWriter{ResponseWriter: w}
		if err := jsengine.Do_Action(rw, r, &jscode); err == nil {
			log.Println("task", r.PathValue("action"), "test - ok")
			if !rw.wroteHeader {
				// 设置正确的Content-Type
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusOK)
			}
		}
	})

	mux.HandleFunc("/player", func(w http.ResponseWriter, r *http.Request) {
		// 从嵌入的文件系统读取html
		content, _ := Assets.ReadFile("website/music.html")
		// 设置正确的Content-Type
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	})
	mux.HandleFunc("/player/last", do_PlayLast)
	mux.HandleFunc("/player/src", do_PlaySrc)
	mux.HandleFunc("/player/play", do_Play)
	mux.HandleFunc("/player/stop", do_Stop)
	mux.HandleFunc("/player/skip", do_Skip)
	mux.HandleFunc("/player/playMode", do_PlayMode)
	mux.HandleFunc("/player/favorited", do_Favorited)
	mux.HandleFunc("/player/set_volume", do_setVolume)
	mux.HandleFunc("/player/get_volume", do_getVolume)

	mux.HandleFunc("/music/list", do_MusicList)
	// 创建Music子文件系统，将共享的music文件系统注册到HTTP服务器
	/*if config.MusicPath != "" {
		mux.Handle("/music/", http.StripPrefix("/music/", http.FileServer(http.Dir(config.MusicPath))))
	}*/
	MusicFS = NewDynamicMusicServer(config.MusicPath)
	mux.Handle("/music/", MusicFS)

	// 注册WS服务路由：ws://localhost:9997/ws
	mux.HandleFunc("/ws", music.WsHandler)

	mux.HandleFunc("/gcron/save", do_Savegcron)
	mux.HandleFunc("/gcron/delete", do_Deletegcron)
	mux.HandleFunc("/gcron/list", do_gcronList)
	mux.HandleFunc("/gcron/setactive", do_setgcronActive)
	mux.HandleFunc("/gcron/getnexttime", do_getNextTime)
	mux.HandleFunc("/gcron/getlunar", do_getLunar)
	mux.HandleFunc("/schedule/script", do_scheduleScript)
	mux.HandleFunc("/schedule/test", do_scheduleTest)
	mux.HandleFunc("/schedule/save", do_scheduleSave)
	//任务内容js编辑
	mux.HandleFunc("/schedule", func(w http.ResponseWriter, r *http.Request) {
		// 从嵌入的文件系统读取html
		content, _ := Assets.ReadFile("website/schedules.html")
		// 设置正确的Content-Type
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	})

	return mux
}

type DynamicMusicServer struct {
	handler atomic.Value // 存储当前http.Handler
}

func NewDynamicMusicServer(initialPath string) *DynamicMusicServer {
	dfs := &DynamicMusicServer{}
	dfs.UpdateHandler(initialPath)
	return dfs
}

// 更新目录时调用此方法
func (dfs *DynamicMusicServer) UpdateHandler(newPath string) {
	// 创建新的FileServer实例
	handler := http.StripPrefix("/music/", http.FileServer(http.Dir(newPath)))
	dfs.handler.Store(handler)
}

// 实现http.Handler接口
func (dfs *DynamicMusicServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := dfs.handler.Load().(http.Handler)
	handler.ServeHTTP(w, r)
}
