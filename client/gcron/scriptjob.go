package gcron

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"ninego/log"
	"os"
	"strings"
	"time"
)

// ScriptExecutor 脚本执行接口
type ScriptExecutor interface {
	Execute(script string, name string) error
}

// ScriptJob 是专门执行脚本的Job实现
type ScriptJob struct {
	Schedule *PeriodSchedule
	Executor ScriptExecutor
}

// Run 执行脚本
func (sj ScriptJob) Run() {
	if sj.Schedule != nil /*&& sj.Schedule.RunScript != ""*/ {
		log.Debugf("ScriptJob执行脚本: %s\n", sj.Schedule.Name)
		err := sj.Schedule.LoadScript()
		if err != nil {
			log.Printf("脚本执行失败 [%s]: %v\n", sj.Schedule.Name, err)
		}
		// 使用指定的执行器执行脚本
		executor := sj.Executor
		if executor != nil {
			err := executor.Execute(sj.Schedule.RunScript, sj.Schedule.Name)
			if err != nil {
				log.Printf("脚本执行失败 [%s]: %v\n", sj.Schedule.Name, err)
			}
		}
	} else {
		log.Printf("ScriptJob: 无脚本可执行 - %s\n", sj.Schedule.Name)
	}
}

// 添加计划任务到Cron
func (c *Cron) Schedule(cronJob *CronJob, executor ScriptExecutor) {
	for _, entry := range c.entries {
		var schedule Schedule
		schedule = cronJob.Schedule
		if schedule == entry.Schedule {
			return
		}
	}
	schedule := Recurring(cronJob)
	cronJob.Schedule = schedule

	// 使用合适的Job接口
	var job Job
	job = ScriptJob{
		Schedule: schedule,
		Executor: executor,
	}
	schedule.Job = job

	// 添加到调度器
	c.Add(schedule, schedule)
}

// 移除计划任务到Cron
func (c *Cron) RemoveSchedule(cronJob *CronJob) {
	if cronJob.Schedule == nil {
		return
	}
	for idx, entry := range c.entries {
		var schedule Schedule
		schedule = cronJob.Schedule
		if schedule == entry.Schedule {
			c.entries = append(c.entries[:idx], c.entries[idx+1:]...)
			break
		}
	}
	cronJob.Schedule = nil
}

// 定时任务
type CronJob struct {
	Filename string `json:"filename"`  //可包括路径的文件名,但不包括文件后缀(.json/.job) clock开头,接流水数字
	IsActive bool   `json:"is_active"` //是否启用 Open/Close

	Name      string    `json:"name"`
	StartTime time.Time `json:"start_time"` //开始日期时间
	Lunar     string    `json:"lunar"`      //农历

	JobCycle       int    `json:"job_cycle"`       //周期类型: 0=一次 1=每日 2=每周 3=每月 -1=每年
	CycleDetails   []int  `json:"cycle_details"`   //周期明细 bits=0->all, bit30..0=1~31日/1~12月/星期1~7 periodic
	SkipHolidays   bool   `json:"skip_holidays"`   //跳过节假日
	SkipWeekdays   bool   `json:"skip_weekdays"`   //跳过工作日
	JobRepeat      bool   `json:"job_repeat"`      //是否重复
	RepeatInterval string `json:"repeat_interval"` //间隔时长
	RepeatDuration string `json:"repeat_duration"` //持续时间(时长表达式,小于1s=次数)

	JobEnd   int       `json:"job_end"`   //0=永久 1=次数 2=日期时间
	EndCount int       `json:"end_count"` //执行次数
	EndDate  time.Time `json:"end_date"`  //终止日期

	Schedule *PeriodSchedule `json:"-"`
}

// 生成调度计划的人性化描述（如“每1,3,5日”、“每周一,三,五”）
func (t *CronJob) ParseScheduleDescription() string {
	var str string
	switch t.JobCycle {
	case 0:
		str = "一次："
		if t.StartTime.Truncate(24*time.Hour) == time.Now().Truncate(24*time.Hour) {
			str += "今天"
		} else //
		if t.StartTime.Truncate(24*time.Hour) == time.Now().Truncate(24*time.Hour).AddDate(0, 0, 1) {
			str += "明天"
		}
		if t.Lunar != "" {
			str += t.Lunar
		} else {
			str += fmt.Sprintf("%d年%02d月%02d日", t.StartTime.Year(), t.StartTime.Month(), t.StartTime.Day())
		}
	case 1:
		str = "每日"
		if len(t.CycleDetails) > 0 {
			if t.Lunar == "" {
				str = "每"
				for _, n := range t.CycleDetails {
					str += fmt.Sprintf("%d,", n)
				}
				str = strings.TrimRight(str, ",")
				str += "日"
			} else {
				str = "每"
				for _, n := range t.CycleDetails {
					str += fmt.Sprintf("%s,", LunarDayStr(n))
				}
				str = strings.TrimRight(str, ",")
			}
		}
	case 2:
		str = "每周"
		if len(t.CycleDetails) > 0 {
			str += "："
			weeks := []string{"一", "二", "三", "四", "五", "周六", "周日"}
			for _, n := range t.CycleDetails {
				str += fmt.Sprintf("%s,", weeks[n-1])
			}
			str = strings.TrimRight(str, ",")
		}
	case 3:
		str = "每月"
		if len(t.CycleDetails) > 0 {
			if t.Lunar == "" {
				str = "每"
				for _, n := range t.CycleDetails {
					str += fmt.Sprintf("%d,", n)
				}
				str = strings.TrimRight(str, ",")
				str += fmt.Sprintf("月%d号", t.StartTime.Day())
			} else {
				str = "每"
				for _, n := range t.CycleDetails {
					str += fmt.Sprintf("%s月,", LunarMonthStr(n))
				}
				str = strings.TrimRight(str, ",")
				lunar := SolarToLunar(t.StartTime)
				str += LunarDayStr(lunar.Day)
			}
		} else {
			if t.Lunar == "" {
				str += fmt.Sprintf("%d号", t.StartTime.Day())
			} else {
				lunar := SolarToLunar(t.StartTime)
				str += LunarDayStr(lunar.Day)
			}
		}
	case -1:
		str = "每年"
		if len(t.CycleDetails) > 0 {
			str += "："
			for _, n := range t.CycleDetails {
				str += solarTerms[n-1] + ","
			}
			str = strings.TrimRight(str, ",")
		} else {
			if t.Lunar == "" {
				str += fmt.Sprintf("：%d月%d日", t.StartTime.Month(), t.StartTime.Day())
			} else {
				lunar := SolarToLunar(t.StartTime)
				str += fmt.Sprintf("：%s月%s", LunarMonthStr(lunar.Month), LunarDayStr(lunar.Day))
			}
		}
	}
	return str
}

func (t *CronJob) Assign(src *CronJob) {
	// 深度复制基础字段
	t.Filename = src.Filename
	t.IsActive = src.IsActive
	t.Name = src.Name
	t.StartTime = src.StartTime
	t.Lunar = src.Lunar
	t.JobCycle = src.JobCycle
	t.CycleDetails = make([]int, len(src.CycleDetails))
	copy(t.CycleDetails, src.CycleDetails)
	t.SkipHolidays = src.SkipHolidays
	t.SkipWeekdays = src.SkipWeekdays
	t.JobRepeat = src.JobRepeat
	t.RepeatInterval = src.RepeatInterval
	t.RepeatDuration = src.RepeatDuration
	t.JobEnd = src.JobEnd
	t.EndCount = src.EndCount
	t.EndDate = src.EndDate
	// 特殊处理Schedule指针
	if t.Schedule != nil {
		Recurring(t)
	}
}

func Load(filepath string) *CronJob {
	// 读取文件内容
	file, err := os.Open(filepath /*+ ".json"*/)
	if err != nil {
		return nil
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil
	}

	// 解析JSON到CronJob结构体
	var cronJob CronJob
	if err := json.Unmarshal([]byte(content), &cronJob); err != nil {
		log.Println(err)
		return nil
	}
	cronJob.Filename = filepath

	return &cronJob
}
func Save(task *CronJob) error {
	// 确定配置文件路径
	taskfile := task.Filename

	if taskfile == "" {
		task.Filename = findNewClockName()
	}

	// 将配置转换为JSON
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		task.Filename = taskfile
		return err
	}

	// 写入文件
	if err := ioutil.WriteFile(task.Filename, data, 0600); err != nil {
		task.Filename = taskfile
		return err
	}

	return nil
}

type PeriodSchedule struct {
	//Task     CronJob //用指针*执行jobCopy.Run()时访问Task报错 ？
	Filename string
	Name     string

	//开始日期时间
	StartTime time.Time
	//农历
	IsLunar bool

	//周期类型
	TaskCycle int //0=一次 1=每日 2=每周 3=每月 -1=每年
	//周期明细
	CycleDetails int //bits=0->all, bit30..0=1~31日/1~12月/星期1~7
	//跳过节假日
	SkipHolidays bool
	//跳过工作日
	SkipWeekdays bool
	//是否重复
	TaskRepeat     bool
	RepeatInterval time.Duration //间隔时长
	RepeatDuration time.Duration //持续时间(时长表达式,小于1s=次数)

	TaskEnd  int //0=永久 1=次数 2=日期时间
	EndCount int
	EndDate  time.Time

	RunScript string
	Job       Job // 任务执行接口

	//下一次开始
	NextTime time.Time
	//是否已过期 isExpired
	isTimesOver bool
	//现在距下次触发时间
	afterPeriod time.Duration

	//needRepeat    bool
	currentRepeat int64 // 当前重复序号（0起始）
	repeatCount   int64 // 要重复的次数
	runTotal      int   //总次数
}

// 生成定时任务
func Recurring(cronJob *CronJob) *PeriodSchedule {
	var schedule *PeriodSchedule
	if cronJob.Schedule == nil {
		schedule = &PeriodSchedule{}

		schedule.RunScript = ""
		schedule.Job = nil
		schedule.runTotal = 0
	} else {
		schedule = cronJob.Schedule
	}
	// 转换CronJob到PeriodSchedule
	schedule.Filename = strings.TrimSuffix(cronJob.Filename, ".json") + ".job"
	schedule.Name = cronJob.Name
	schedule.StartTime = cronJob.StartTime
	schedule.IsLunar = cronJob.Lunar != ""
	schedule.TaskCycle = cronJob.JobCycle
	schedule.SkipHolidays = cronJob.SkipHolidays
	schedule.SkipWeekdays = cronJob.SkipWeekdays
	schedule.TaskRepeat = cronJob.JobRepeat
	schedule.TaskEnd = cronJob.JobEnd
	schedule.EndCount = cronJob.EndCount
	schedule.EndDate = cronJob.EndDate
	//schedule.RunScript = ""
	//schedule.Job = nil // 传递CronJob中的Job接口
	schedule.isTimesOver = false
	schedule.repeatCount = 0
	//schedule.runTotal = 0

	// 转换CycleDetails []int -> int
	if len(cronJob.CycleDetails) > 0 {
		bits := 0
		for _, bit := range cronJob.CycleDetails {
			if bit == 0 { // 0 表示全选
				bits = 0
				break
			}
			bits |= 1 << (bit - 1)
		}
		schedule.CycleDetails = bits
	}

	// 解析时间间隔
	if cronJob.JobRepeat {
		if dur, err := ParseDuration(cronJob.RepeatInterval); err == nil {
			schedule.RepeatInterval = dur
		}
		if dur, err := ParseDuration(cronJob.RepeatDuration); err == nil || err == ErrTimesValid {
			schedule.RepeatDuration = dur
		}
	}

	/*/ 读取脚本文件
	filepath := cronJob.Filename //+ ".job"
	if IsExist(filepath) {
		file, err := os.Open(filepath)
		if err != nil {
			return nil
		}
		defer file.Close()
		script, _ := ioutil.ReadAll(file)
		schedule.RunScript = string(script)
	}
	fmt.Printf("schedule=%+v\n", schedule)*/
	return schedule
}

func (p *PeriodSchedule) nextAfter(next, t time.Time) bool {
	if !next.After(t) {
		return false
	}
	if p.SkipHolidays { //跳过节假日
		if IsHoliday(next) {
			//log.Debug("跳过节假日", next)
			return false
		}
	}
	if p.SkipWeekdays { //跳过工作日
		if IsWorkday(next) {
			//log.Debug("跳过工作日", next)
			return false
		}
	}
	return true
}

func (p *PeriodSchedule) reset(t time.Time) time.Time {
	var nextTime time.Time
	yy, mm, dd := 0, 0, 0
	IsLunar := p.IsLunar
	switch p.TaskCycle {
	case 0: //一次（不用考滤任何其他条件/除非过期）
		nextTime = p.StartTime
		if /*!nextTime.After(t)*/ !p.nextAfter(nextTime, t) {
			nextTime = t
		}
		return nextTime
	case -1: //每年
		yy = 1
		if p.CycleDetails != 0 {
			IsLunar = false
		}
	case 1: //每日
		dd = 1
	case 2: //每周
		dd = 1
		IsLunar = false
	case 3: //每月
		mm = 1
	}
	year, month, day := p.StartTime.Date()
	hour, min, sec := p.StartTime.Clock()
	if IsLunar {
		//当前时间转农历year/month/day
		lunar := SolarToLunar(t)
		year = lunar.Year
		month = time.Month(lunar.Month)
		day = lunar.Day
		//得到开始农历
		lunar = SolarToLunar(p.StartTime)
		lunar.Year = year
		switch p.TaskCycle {
		case -1: //每年
			//每年->当前农历年+开始农历月日
		case 1: //每日
			//每日->当前农历年月日
			lunar.Month = int(month)
			lunar.Day = day
		case 3: //每月
			//每月->当前农历年月+开始农历日
			lunar.Month = int(month)
		}
		//古人认为：闰月是“补出来的月份”，不是正式月份!
		lunar.IsLeap = false
		//农历->公历
		nextTime = LunarToSolar(lunar, p.StartTime)
		for /*!nextTime.After(t)*/ !p.nextAfter(nextTime, t) {
			year += yy
			month += time.Month(mm)
			day += dd
			//加1后的农历转公历
			nextTime = LunarBuildToSolar(year, int(month), day, p.StartTime)
		}
		if p.CycleDetails != 0 {
			key := 0
			for (p.CycleDetails&dayToBitMask(key)) == 0 || !p.nextAfter(nextTime, t) {
				if key != 0 {
					year += yy
					month += time.Month(mm)
					day += dd
					//加1后的农历转公历
					nextTime = LunarBuildToSolar(year, int(month), day, p.StartTime)
				}
				switch p.TaskCycle {
				case 1: //每日
					key = day
				case 3: //每月
					key = int(month)
				}
			}
		}
	} else {
		if p.TaskCycle == -1 {
			year, _, _ = t.Date()
		} else if p.TaskCycle == 3 {
			year, month, _ = t.Date()
		} else {
			year, month, day = t.Date()
		}
		nextTime = time.Date(year, month, day, hour, min, sec, 0, time.Local)
		for /*!nextTime.After(t)*/ !p.nextAfter(nextTime, t) {
			nextTime = nextTime.AddDate(yy, mm, dd)
		}
		if p.CycleDetails != 0 {
			if p.TaskCycle == -1 { //每年
				//重算当前年份
				year, month, day = t.Date()
				nextTime = time.Date(year, month, day, hour, min, sec, 0, time.Local)
				if !nextTime.After(t) {
					nextTime = nextTime.AddDate(0, 0, 1)
				}
				//得到当前向后最近的节气
			nextterm:
				term := GetSolarTermByDate(nextTime)
				for term == "" {
					nextTime = nextTime.AddDate(0, 0, 1)
					term = GetSolarTermByDate(nextTime)
				}
				var Start int
				for i := 0; i < 24; i++ {
					if solarTerms[i] == term {
						Start = i //查节气solarTerms中的索引index
						break
					}
				}
				//检查是否在选中节气列表中
				key := Start + 1
				if (p.CycleDetails&dayToBitMask(key)) == 0 || !p.nextAfter(nextTime, t) {
					nextTime = nextTime.AddDate(0, 0, 14) //相邻节气的最小间隔约14.7天（精确值14天17小时），取整可按 14 天处理
					goto nextterm                         //没选中,+14向下一节气
				}
			} else {
				key := 0
				for (p.CycleDetails&dayToBitMask(key)) == 0 || !p.nextAfter(nextTime, t) {
					if key != 0 {
						nextTime = nextTime.AddDate(yy, mm, dd) // 增加
					}
					year, month, day = nextTime.Date()
					switch p.TaskCycle {
					case 1: //每日
						key = day
					case 2: //每周
						key = int(nextTime.Weekday())
						if key == 0 {
							key = 7
						}
					case 3: //每月
						key = int(month)
					}
				}
			}
		}
	}
	return nextTime
}

func (p *PeriodSchedule) Run() {
	if p.repeatCount > 0 {
		p.currentRepeat += 1
		if p.TaskEnd == 1 && p.currentRepeat == 1 { //次数
			p.runTotal += 1
		}
	} else {
		if p.TaskEnd == 1 { //次数
			p.runTotal += 1

		}
	}

	log.Printf("执行任务: %s, 当前重复次数: %d, 总执行次数: %d. time=%s\n", p.Name, p.currentRepeat, p.runTotal, time.Now())

	if p.TaskEnd == 1 { //次数
		if p.runTotal > p.EndCount {
			p.isTimesOver = true
		}
	}
	if p.TaskEnd == 2 { //时间
		if p.NextTime.After(p.EndDate) {
			p.isTimesOver = true
		}
	}

	/*/if p.RunScript != "" {
		err := p.loadScript()
		if err != nil {
			fmt.Printf("任务执行失败 [%s]: %v\n", p.Name, err)
		} else {
			fmt.Printf("任务执行成功 [%s]\n", p.Name)
		}
	}*/
	// 实际执行脚本
	if p.Job != nil {
		p.Job.Run()
	}

}

// loadScript 执行任务脚本
func (p *PeriodSchedule) LoadScript() error {
	if p.RunScript == "" {
		if IsExist(p.Filename) {
			file, err := os.Open(p.Filename)
			if err != nil {
				return err
			}
			defer file.Close()
			script, _ := ioutil.ReadAll(file)
			p.RunScript = string(script)
		}
	}
	if p.RunScript == "" {
		return fmt.Errorf("脚本内容为空")
	}
	return nil
}

func (p *PeriodSchedule) nextPeriod(t time.Time) time.Duration {
	// 如果任务已经过期，返回0表示不再执行
	if p.isTimesOver {
		return 0
	}

	var nextTime time.Time

	// 处理重复执行逻辑
	// 修复：重新组织重复执行的逻辑，使其更清晰易懂
	if p.repeatCount > 0 {
		// 当前正在重复执行中
		log.Debugf("重复执行中: currentRepeat=%d, repeatCount=%d, 当前时间=%v, 下次时间=%v\n",
			p.currentRepeat, p.repeatCount, t.Truncate(time.Second), p.NextTime.Truncate(time.Second))

		// 检查是否在重复执行的时间窗口内
		if p.currentRepeat <= p.repeatCount {
			// 是否是由after触发(t=next)
			if t.Truncate(time.Second).Equal(p.NextTime.Truncate(time.Second)) {
				// 计算下一次重复执行的时间
				p.currentRepeat++
			}
			// NextTime为下一次重复执行的时间
			nextTime = p.NextTime.Add(p.RepeatInterval * time.Duration(p.currentRepeat))
			p.afterPeriod = nextTime.Sub(t)
			log.Debugf("下一次重复执行: %v, 间隔: %v\n", nextTime, p.afterPeriod)
			return p.afterPeriod
		}

		// 重复执行完成，重置状态
		log.Println("重复执行完成，重置状态")
		p.currentRepeat = 0
		p.repeatCount = 0
	}

	// 检查是否需要重复执行
	if p.TaskRepeat {
		// 计算重复次数
		log.Debugf("计算重复次数: RepeatDuration=%v, RepeatInterval=%v\n",
			int64(p.RepeatDuration), int64(p.RepeatInterval))

		if p.RepeatDuration < time.Second {
			// RepeatDuration小于1秒，表示具体的重复次数
			p.repeatCount = int64(p.RepeatDuration)
		} else if p.RepeatInterval > 0 {
			// 根据持续时间和间隔计算重复次数
			p.repeatCount = int64(p.RepeatDuration / p.RepeatInterval)
		} else {
			// 默认重复1次
			p.repeatCount = 1
		}

		// 确保至少重复1次
		if p.repeatCount < 1 {
			p.repeatCount = 1
		}

		// 重置当前重复序号
		p.currentRepeat = 0
		log.Debugf("设置重复次数: %d\n", p.repeatCount)
	}

	// 计算下一次周期的开始时间
	nextTime = p.reset(t)

	// 更新状态
	p.NextTime = nextTime
	p.afterPeriod = nextTime.Sub(t)

	log.Debugf("下一次执行: %v, 间隔: %v\n", nextTime, p.afterPeriod)
	return p.afterPeriod
}

// IsExpired 检查任务是否已经过期（所有执行次数已完成或已超过结束时间）
// 返回true表示任务已经完成，不再需要调度
func (p *PeriodSchedule) IsExpired() bool {
	return p.isTimesOver
}

// Next adds time t to underlying period
func (ps *PeriodSchedule) Next(t time.Time) time.Time {
	//return t.Truncate(time.Second).Add(ps.nextPeriod(t))
	//把周期加到当前的 time.Time 上，返回新的时间点。
	return t.Add(ps.nextPeriod(t))
}

// 将月份中的日期号数（1-31）转换为位掩码值
// 示例：1号 -> 1 (0b0001), 2号 -> 2 (0b0010), 3号 -> 4 (0b0100)
func dayToBitMask(day int) int {
	if day < 1 || day > 31 {
		return 0 // 无效日期返回0
	}
	return 1 << (day - 1) // 关键位运算：左移day-1位
}
