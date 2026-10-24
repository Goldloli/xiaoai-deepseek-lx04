package webui

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"ninego/log"
	"os"
	"strings"
	"time"
	"xiaobot/gcron"
	"xiaobot/jsengine"
)

type TaskItem struct {
	gcron.CronJob
	Description string    `json:"description"` //调度计划的中文描述
	IsExpired   bool      `json:"is_expired"`  //CronJob.Schedule.isTimesOver = false
	Next        time.Time `json:"next"`        //CronJob.Schedule.NextTime
}

func do_Savegcron(writer http.ResponseWriter, request *http.Request) {
	log.Debug("save...")
	var item TaskItem
	err := json.NewDecoder(request.Body).Decode(&item)
	if err != nil {
		log.Error("Error:", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	item.StartTime = item.StartTime.Local()
	item.EndDate = item.EndDate.Local()
	//jsonBytes, _ := json.Marshal(item)
	//log.Degut("item=", string(jsonBytes))

	//保存
	isNewfile := (item.Filename == "")
	if err := gcron.Save(&item.CronJob); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if isNewfile {
		jsengine.Schedules.Add(&item.CronJob)
		//item.Filename = item.CronJob.Filename
		//item.IsExpired = true
	} else {
		jsengine.Schedules.Update(&item.CronJob)
		if item.CronJob.Schedule != nil {
			item.Next = item.CronJob.Schedule.NextTime
		}
	}
	item.Description = item.ParseScheduleDescription()

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	//writer.Write([]byte(`{"filename":"` + item.Filename + `"}`))
	json.NewEncoder(writer).Encode(item)
}

func do_Deletegcron(writer http.ResponseWriter, request *http.Request) {
	// 解析查询参数
	filename := request.URL.Query().Get("filename")
	if filename == "" {
		http.Error(writer, "任务名称为空", http.StatusBadRequest)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	task := jsengine.Schedules.Jobs[filename]
	if task != nil {
		fnjob := strings.TrimSuffix(filename, ".json") + ".job"
		if gcron.IsExist(fnjob) {
			err := os.Remove(fnjob)
			if err != nil {
				writer.Write([]byte(fmt.Sprintf(`{"msg":"%s","success":0}`, err.Error())))
				return
			}
		}
	}
	err := os.Remove(filename)
	if err != nil {
		writer.Write([]byte(fmt.Sprintf(`{"msg":"%s","success":0}`, err.Error())))
		return
	}
	jsengine.Schedules.Remove(filename)
	writer.Write([]byte(`{"msg":"","success":1}`))
}

func do_gcronList(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	list := jsengine.Schedules.List()
	if len(list) == 0 {
		writer.Write([]byte(`[]`))
		return
	}
	var items []TaskItem
	for _, task := range list {
		item := TaskItem{CronJob: *task}
		item.Description = task.ParseScheduleDescription()
		if task.Schedule != nil {
			item.IsActive = true
			item.Next = task.Schedule.NextTime
		}
		//item.IsExpired = false
		items = append(items, item)
	}
	//jsonBytes, _ := json.Marshal(items)
	//log.Println("items:", string(jsonBytes))
	json.NewEncoder(writer).Encode(items)
}

// 公历转农历
func do_getLunar(writer http.ResponseWriter, request *http.Request) {
	jsTimeStr := request.URL.Query().Get("time")
	t, err := time.Parse(time.RFC3339, jsTimeStr)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	t = t.Local()
	lunar := gcron.SolarToLunar(t)
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(fmt.Sprintf(`{"lunar":"%s"}`, lunar.Str)))
}

// 设置计划启用
func do_setgcronActive(writer http.ResponseWriter, request *http.Request) {
	// 解析查询参数
	filename := request.URL.Query().Get("filename")
	active := request.URL.Query().Get("active")
	flag := 0
	if active == "1" {
		flag = 1
	}
	flag = jsengine.Schedules.SetActive(filename, flag)
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(fmt.Sprintf(`{"is_active":%d}`, flag)))
}

func do_getNextTime(writer http.ResponseWriter, request *http.Request) {
	// 解析查询参数
	filename := request.URL.Query().Get("filename")
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	task := jsengine.Schedules.Jobs[filename]
	rest := struct {
		Next      time.Time `json:"next"`
		IsExpired bool      `json:"is_expired"`
		AlarmTime time.Time `json:"alarm_time"` //下次触发的时间点
	}{
		//Next:      task.Schedule.NextTime,
		IsExpired: true,
		AlarmTime: jsengine.Schedules.Cron.Effective,
	}
	if rest.AlarmTime.Year() > time.Now().Year()+5 {
		rest.AlarmTime = time.Time{}
	}
	if task != nil && task.Schedule != nil {
		rest.Next = task.Schedule.NextTime
		rest.IsExpired = !task.Schedule.NextTime.After(time.Now())
	}
	log.Debugf("do_getNextTime=%+v\n", rest)
	json.NewEncoder(writer).Encode(rest)
}

func do_scheduleScript(writer http.ResponseWriter, request *http.Request) {
	filename := request.URL.Query().Get("filename")
	log.Debug("script=", filename)
	// 设置正确的Content-Type
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	task := jsengine.Schedules.Jobs[strings.TrimSuffix(filename, ".job")+".json"]
	if task != nil && task.Schedule != nil {
		task.Schedule.LoadScript()
		writer.Write([]byte(task.Schedule.RunScript))
	} else {
		var script []byte
		if gcron.IsExist(filename) {
			file, err := os.Open(filename)
			if err == nil {
				script, _ = ioutil.ReadAll(file)
				file.Close()
			}
		}
		writer.Write(script)
	}
}

func do_scheduleTest(writer http.ResponseWriter, request *http.Request) {
	filename := request.URL.Query().Get("filename")
	// 更安全的读取方式
	body, err := io.ReadAll(request.Body)
	if err != nil {
		log.Error("Read body failed:", err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	defer request.Body.Close() // 确保关闭body
	jscode := string(body)
	// 执行
	exec := jsengine.DefaultScriptExecutor{}
	if err := exec.Execute(jscode, filename); err != nil {
		log.Error("Run jscript failed:", err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("Run", filename, "test - ok")
	// 设置正确的Content-Type
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte("success"))
}

func do_scheduleSave(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || request.Body == nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	filename := request.URL.Query().Get("filename")
	// 更安全的读取方式
	body, err := io.ReadAll(request.Body)
	if err != nil {
		log.Error("Read body failed:", err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	defer request.Body.Close() // 确保关闭body
	err = ioutil.WriteFile(filename, body, 0666)
	if err != nil {
		log.Errorf("Write file '%s' failed:%s", filename, err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("Write file", filename, " - ok")
	task := jsengine.Schedules.Jobs[strings.TrimSuffix(filename, ".job")+".json"]
	if task == nil {
	} else if task.Schedule != nil {
		task.Schedule.RunScript = string(body)
	}
	// 设置正确的Content-Type
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
}
