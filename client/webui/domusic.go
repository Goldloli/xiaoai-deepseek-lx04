package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ninego/log"
	"path/filepath"
	"strconv"
	"strings"
	"xiaobot/music"
)

func do_MusicList(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	dirParam := r.URL.Query().Get("dir")
	if dirParam == "history" {
		log.Debug("history", music.HistoryMusicFiles)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		music.CheckFavorited(&music.HistoryMusicFiles)
		json.NewEncoder(w).Encode(music.HistoryMusicFiles)
	} else if dirParam == "favorite" {
		log.Debug("favorite", music.FavoritedMusicFiles)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(music.FavoritedMusicFiles)
	} else {
		music.PlayerMutex.Lock()
		music.PlayerState.CurrentMusicDir = dirParam
		//music.PlayerState.FilteredMusicFiles = []music.FileItem{}
		music.PlayerMutex.Unlock()

		dirParam = strings.TrimPrefix(dirParam, "music")
		filter := r.URL.Query().Get("filter")
		currentDir := filepath.Clean(dirParam) // 路径安全处理
		// 组合完整路径（假设config.MusicPath是基础目录）
		fullPath := filepath.Join(config.MusicPath, currentDir)
		// 安全验证：防止路径遍历攻击
		if !strings.HasPrefix(fullPath, config.MusicPath) {
			http.Error(w, "Invalid directory path", http.StatusBadRequest)
			return
		}
		files, _ := music.FindAudioFiles(config.MusicPath, currentDir, filter)
		log.Debug("List Music files =", files)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		music.CheckFavorited(&files)
		json.NewEncoder(w).Encode(files)
	}
}

func do_PlayLast(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(music.PlayerState)
	log.Debug("music.PlayerState=>", music.PlayerState)
}

func do_PlaySrc(w http.ResponseWriter, r *http.Request) {
	music.PlayerMutex.Lock()
	err := json.NewDecoder(r.Body).Decode(&music.PlayerState)
	music.PlayerMutex.Unlock()
	if err != nil {
		log.Error("request Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
	log.Debug("music.PlayerState<=", music.PlayerState)
}

func do_Favorited(w http.ResponseWriter, r *http.Request) {
	var item music.FileItem
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		log.Error("request Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	music.ToFavorited(item)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
	log.Debug("music.Favorited =", item)
}

func do_PlayMode(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if mode != "" {
		music.SetPlayMode(mode)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"mode": "%s"}`, music.PlayerState.PlayMode)))
}

func do_Play(w http.ResponseWriter, r *http.Request) {
	err := music.Play(bot, config)
	if err != nil {
		log.Error("Player Stop Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}
func do_Stop(w http.ResponseWriter, r *http.Request) {
	err := music.Stop(bot)
	if err != nil {
		log.Error("Player Stop Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func do_Skip(w http.ResponseWriter, r *http.Request) {
	value := r.URL.Query().Get("value")
	i, _ := strconv.Atoi(value)
	if music.SkipChannel != nil {
		music.SkipChannel.C <- i //SkipChannel 时需传入正整数+1（下一首）/ 负整数-1（上一首）
	}
	w.WriteHeader(http.StatusOK)
}

func do_setVolume(w http.ResponseWriter, r *http.Request) {
	vol := r.URL.Query().Get("value")
	i, err := strconv.Atoi(vol)
	if err != nil {
		log.Error("Player Volume Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := music.SetVolume(bot, i); err != nil {
		log.Error("Player SetVolume Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	i = music.GetVolume(bot)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"volume": %d}`, i)))
}

func do_getVolume(w http.ResponseWriter, r *http.Request) {
	vol := music.GetVolume(bot)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"volume": %d}`, vol)))
}
