package jsengine

import (
	"fmt"
	"net/http"
	"ninego/log"
	"os"
	"path/filepath"
	"sort"
	"time"
	"xiaobot/gcron"

	"github.com/dop251/goja"
)

// DefaultScriptExecutor 默认脚本执行器
type DefaultScriptExecutor struct{}

// Execute 执行脚本
func (e DefaultScriptExecutor) Execute(script string, name string) error {
	defer func() {
		global.Update("runcount", -1)
		if r := recover(); r != nil {
			log.Error("Recovered from panic:", r)
		}
	}()
	global.Update("runcount", +1)

	// 创建 JS 虚拟机
	engine := vmPool.Get().(*Engine)
	// 将 VM 放回池中以供将来重用
	defer vmPool.Put(engine)

	// 将 Go 对象注入 JS 全局作用域
	//engine.Runtime.Set("timestr", timeStr)

	// 执行 JS 脚本，处理错误
	_, err := engine.RunString("{" + script + "\n}")
	if err != nil {
		if evalErr, ok := err.(*goja.Exception); ok {
			log.Error("JS 脚本错误: "+evalErr.String(), http.StatusInternalServerError)
			return evalErr
		}
		log.Error("JS 执行失败: "+err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

var Schedules = &schedules{
	Jobs: make(map[string]*gcron.CronJob),
	Cron: gcron.New(),
}

type schedules struct {
	Jobs map[string]*gcron.CronJob
	Cron *gcron.Cron
}

func (s *schedules) Update(task *gcron.CronJob) {
	job := s.Jobs[task.Filename]
	if job == nil {
		log.Printf("错误：找不到任务 %s\n", task.Filename)
		return
	}

	// 修复：更新任务配置
	job.Assign(task)
	log.Printf("更新任务: %+v\n", s.Jobs[task.Filename])

	// 如果任务有调度计划，需要重置调度器
	if job.Schedule != nil {
		log.Debug("update - reset")
		s.Cron.Reset(job.Schedule)
	}
}

func (s *schedules) Add(task *gcron.CronJob) {
	// 添加任务到任务列表
	s.Jobs[task.Filename] = task

	// 修复：只有当任务启用且没有调度计划时，才创建调度
	// 原来的逻辑是 task.Schedule == nil，这是错误的
	// 应该是：如果任务启用，就创建调度计划
	if task.IsActive {
		log.Printf("添加任务并创建调度: %s\n", task.Name)
		s.Cron.Schedule(task, DefaultScriptExecutor{})
		s.Cron.Reset(task.Schedule)
	} else {
		log.Printf("添加任务但不创建调度（任务未启用）: %s\n", task.Name)
	}
}

func (s *schedules) Remove(filename string) {
	job := s.Jobs[filename]
	if job == nil {
		return
	}
	if job.Schedule != nil {
		needReset := job.Schedule.NextTime.Equal(s.Cron.Effective)
		s.Cron.RemoveSchedule(job)
		if needReset {
			s.Cron.Reset(nil)
		}

	}
	delete(s.Jobs, filename)
}

// sortMapByTime()
func (s *schedules) List() []*gcron.CronJob {
	var pairs []*gcron.CronJob
	for _, v := range s.Jobs {
		pairs = append(pairs, v)
	}

	// 按值升序排序
	sort.Slice(pairs, func(i, j int) bool {
		//return pairs[i].Value < pairs[j].Value
		hi, mi, _ := pairs[i].StartTime.Clock()
		hj, mj, _ := pairs[j].StartTime.Clock()
		return fmt.Sprintf("%2d%2d", hi, mi) < fmt.Sprintf("%2d%2d", hj, mj)
	})

	return pairs
}

func (s *schedules) SetActive(id string, flag int) int {
	job := s.Jobs[id]
	if job == nil {
		log.Printf("错误：找不到任务 %s\n", id)
		return 0
	}

	log.Printf("设置任务状态: %s, flag=%d\n", job.Name, flag)

	if flag == 0 {
		// 0=关闭任务
		if job.Schedule != nil {
			log.Printf("关闭任务: %s\n", job.Name)
			s.Cron.RemoveSchedule(job)
		}
	} else {
		// 1=打开任务
		if job.Schedule != nil {
			// 任务已经开启，无需重复操作
			log.Printf("任务已开启: %s\n", job.Name)
		} else {
			// 任务未开启，需要调度
			log.Printf("开启任务: %s\n", job.Name)
			s.Cron.Schedule(job, DefaultScriptExecutor{})
		}
	}

	// 更新任务状态
	job.IsActive = (flag == 1)

	// 保存任务配置
	if err := gcron.Save(job); err != nil {
		log.Printf("保存任务失败: %v\n", err)
	}

	// 修复：只在任务状态改变时才重置调度器
	// 并且检查任务是否有效
	log.Debug("setActive - Reset")
	s.Cron.Reset(job.Schedule)

	// 返回任务状态
	if job.Schedule == nil {
		// 任务没有调度计划
		return 0
	} else {
		// 检查任务是否已过期
		// 修复：job.Schedule 是 *PeriodSchedule 类型，可以直接调用方法
		if job.Schedule.IsExpired() {
			log.Printf("任务已过期: %s\n", job.Name)
			return 0
		}
		// 检查下次执行时间是否在未来
		if job.Schedule.NextTime.Before(time.Now()) {
			log.Printf("任务下次执行时间已过: %s, NextTime=%v\n", job.Name, job.Schedule.NextTime)
			return 0
		}
		// 任务有效
		return 1
	}
}

func (s *schedules) FetchFromExecDir() error {
	// 使用Glob函数匹配所有.json文件，支持通配符
	filePaths, err := filepath.Glob(GetExecutableDir() + "/clock*.json")
	if err == nil {
		// 遍历匹配到的文件
		for _, filePath := range filePaths {
			// 获取文件名（不含路径）
			filename := filepath.Base(filePath)
			task := gcron.Load(filename)
			if task == nil {
				continue
			}
			s.Jobs[task.Filename] = task //s.Add(task)
			log.Println(filePath)
		}
		log.Debug("fetch jobs ->", len(s.Jobs), s.Jobs)
	}
	return err
}

func (s *schedules) Start() {
	for _, task := range s.Jobs {
		if task.IsActive {
			s.Cron.Schedule(task, DefaultScriptExecutor{})
		}
	}
	s.Cron.Start()
}

func (s *schedules) Stop() {
	s.Cron.Stop()
}

// GetExecutableDir 获取当前运行程序所在的目录
func GetExecutableDir() string {
	// 获取当前程序的可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	// 从可执行文件路径中提取目录部分
	exeDir := filepath.Dir(exePath)
	return exeDir
}
