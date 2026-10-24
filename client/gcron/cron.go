package gcron

import (
	"ninego/log"
	"reflect"
	"sort"
	"sync"
	"time"
)

/*Cron 里面的 []*Entry 其实就代表了一组【定时任务】，每个【定时任务】可以简化理解为 <触发器，任务> 组成的一个 tuple。
Entry 结构还包含了【前一次执行时间点】和【下一次执行时间点】，这个目前可以忽略，只是为了辅助代码用。*/
// Entry consists of a schedule and the job to be executed on that schedule.
type Entry struct {
	Schedule Schedule
	Job      Job

	// the next time the job will run. This is zero time if Cron has not been
	// started or invalid schedule.
	Next time.Time

	// the last time the job was run. This is zero time if the job has not been
	// run.
	Prev time.Time
}

/*这里是对 Entry 列表的简单封装，因为我们可能同时有多个 Entry 需要调度，处理的顺序很重要。
这里实现了 sort 的接口, 有了 Len(), Swap(), Less() 我们就可以用 sort.Sort() 来排序了。
此处的排序策略是按照时间大小。*/
// byTime is a handy wrapper to chronologically sort entries.
type byTime []*Entry

func (b byTime) Len() int      { return len(b) }
func (b byTime) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// Less reports `earliest` time i should sort before j.
// zero time is not `earliest` time.
func (b byTime) Less(i, j int) bool {
	// 处理 Next == Prev 的情况(所有Next == Prev的数据均会被排在最后)
	if b[i].Next == b[i].Prev {
		return false // i 不排在 j 前
	}
	if b[j].Next == b[j].Prev {
		return true // i 强制排在 j 前
	}

	// 处理零值时间
	if b[i].Next.IsZero() {
		return false
	}
	if b[j].Next.IsZero() {
		return true
	}

	// 正常比较时间
	return b[i].Next.Before(b[j].Next)
}

// Job is the interface that wraps the basic Run method.
//
// Job 是对【任务】的抽象，只需要实现一个 Run 方法，没有入参出参。
//
// Run executes the underlying func.
type Job interface {
	Run()
}

// Cron provides a convenient interface for scheduling job such as to clean-up
// database entry every month.
/*
entries 就是定时任务的核心能力，它记录了一组【定时任务】；
running 用来标识这个 Cron 是否已经启动；
add 是一个channel，用来支持在 Cron 启动后，新增的【定时任务】；
stop 同样是个channel，注意到是空结构体，用来控制 Cron 的停止。这个其实是经典写法了，对日常开发也有借鉴意义

修复：添加互斥锁保护entries的并发访问，防止数据竞争*/
//
// Cron keeps track of any number of entries, invoking the associated func as
// specified by the schedule. It may also be started, stopped and the entries
// may be inspected.
type Cron struct {
	entries []*Entry
	running bool
	add     chan *Entry
	stop    chan struct{}
	reset   chan interface{}

	Effective time.Time

	// 修复：添加互斥锁保护entries的并发访问
	mu sync.RWMutex
}

// New instantiates new Cron instant c.
func New() *Cron {
	//调用 gron.New() 方法后，得到的是一个指向 Cron 对象的指针。此时只是初始化了 stop 和 add 两个 channel，没有启动调度。
	return &Cron{
		stop:  make(chan struct{}),
		add:   make(chan *Entry),
		reset: make(chan interface{}),
	}
}

// Start 方法执行的时候会先将 running 变量置为 true，用来标识实例已经启动
// （启动前后加入的定时任务 Entry 处理策略是不同的，所以这里需要标识），然后启动一个 goroutine 来实际跑启动的逻辑。
// Start signals cron instant c to get up and running.
func (c *Cron) Start() {
	c.running = true
	go c.run()
}

// Add appends schedule, job to entries.
//
// 若 Cron 实例还没启动，加入到 Cron 的 entries 列表里就ok，随后启动的时候会处理。
// 但如果已经启动了，就直接往 add 这个 channel 中塞，走额外的新增调度路径。
//
// if cron instant is not running, adding to entries is trivial.
// otherwise, to prevent data-race, adds through channel.
// 修复：添加锁保护entries的并发访问
func (c *Cron) Add(s Schedule, j Job) {

	entry := &Entry{
		Schedule: s,
		Job:      j,
	}

	if !c.running {
		c.mu.Lock()
		c.entries = append(c.entries, entry)
		c.mu.Unlock()
		return
	}
	c.add <- entry
}

// AddFunc registers the Job function for the given Schedule.
func (c *Cron) AddFunc(s Schedule, j func()) {
	c.Add(s, JobFunc(j))
}

// Stop 方法则会将 running 置为 false，然后直接往 stop channel 塞一个空结构体即可。
// Stop halts cron instant c from running.
func (c *Cron) Stop() {

	if !c.running {
		return
	}
	c.running = false
	c.stop <- struct{}{}
}

func (c *Cron) Reset(s Schedule) {
	if !c.running {
		c.Start()
		return
	}
	if s != nil && !reflect.ValueOf(s).IsNil() {
		next := s.Next(time.Now().Local())
		c.mu.RLock()
		isLock := true
		for _, entry := range c.entries {
			if entry.Schedule == s {
				if next.Before(c.Effective) || (entry.Next.Equal(c.Effective) && !next.Equal(entry.Next)) {
					entry.Next = next //entry.Schedule.Next(now) //得到新的Next时间点
					c.mu.RUnlock()
					isLock = false
					c.reset <- 1
				}
				break
			}
		}
		if isLock {
			c.mu.RUnlock()
		}
		//}
	} else {
		c.reset <- 1
	}
}

var after = time.After

// run the scheduler...是如何把上面 Cron, Entry, Schedule, Job 串起来的。
/*
首先拿到 local 的时间 now；
遍历所有 Entry，调用 Next 方法拿到各个【定时任务】下一次运行的时间点；
对所有 Entry 按照时间排序（我们上面提过的 byTime）；
拿到第一个要到期的时间点，在 select 里面通过 time.After 来监听。
到点了就起动新的 goroutine 跑对应 entry 里的 Job，并回到 for 循环，继续重新 sort，再走同样的流程；
若 add channel 里有新的 Entry 被加进来，就加入到 Cron 的 entries 里，触发新的 sort；
若 stop channel 收到了信号，就直接 return，结束执行。*/
//
// It needs to be private as it's responsible of synchronizing a critical
// shared state: `running`.
// 修复：添加锁保护entries的并发访问
func (c *Cron) run() {

	var effective time.Time
	now := time.Now().Local()

	// 修复：添加锁保护entries的并发访问
	c.mu.Lock()
	// to figure next trig time for entries, referenced from now
	for _, e := range c.entries {
		e.Next = e.Schedule.Next(now)
	}
	c.mu.Unlock()

	for {
		c.mu.Lock()
		sort.Sort(byTime(c.entries))
		c.mu.Unlock()

		// 计算下一次需要执行的时间点（effective）
		// 修复：找到所有entries中最早的有效执行时间
		c.mu.RLock()
		if len(c.entries) > 0 {
			//effective = c.entries[0].Next
			// 遍历所有entries，找到最早的Next时间
			for _, e := range c.entries {
				// 找到第1个Next时间比now晚的entry，更新effective
				effective = e.Next
				if effective.After(now) {
					break
				}
				// Next时间比now早的
				e.Prev = e.Next //相当于标识为无效
			}
		}
		c.mu.RUnlock()

		log.Println("effective => ", effective)

		// 双重检查：确保effective在未来
		if effective.Sub(now) <= 0 {
			log.Println("警告：effective时间在过去，无计划任务可执行")
			effective = now.AddDate(15, 0, 0)
		}

		c.Effective = effective
		select {
		case now = <-after(effective.Sub(now)):
			// 锁保护entries的并发访问
			c.mu.Lock()
			// 移除失效entries
			// 所有Next == Prev的数据均被排在最后
			for i := len(c.entries) - 1; i >= 0; i-- {
				entry := c.entries[i]
				if entry.Next != entry.Prev { //已越过无效区,不再
					break
				}
				log.Debug("移除过期的entry", i)
				c.entries = append(c.entries[:i], c.entries[i+1:]...)
			}

			// 执行所有到期的任务
			// entries with same time gets run. 把时间一样的都goRun()跑起来
			for _, entry := range c.entries {
				if entry.Next != effective {
					break
				}
				entry.Prev = now
				entry.Next = entry.Schedule.Next(now) //得到新的Next时间点
				//go entry.Job.Run()
				jobCopy := entry.Job
				go jobCopy.Run()
			}
			c.mu.Unlock()
		case e := <-c.add:
			e.Next = e.Schedule.Next(time.Now())
			c.mu.Lock()
			c.entries = append(c.entries, e)
			c.mu.Unlock()
		case <-c.reset:
			// nothing...
		case <-c.stop:
			return // terminate go-routine.
		}
		now = time.Now().Local() //修正时间
	}
	c.running = false
}

// Entries returns cron etn
func (c Cron) Entries() []*Entry {
	return c.entries
}

// JobFunc is an adapter to allow the use of ordinary functions as gron.Job
// If f is a function with the appropriate signature, JobFunc(f) is a handler
// that calls f.
//
// todo: possibly func with params? maybe not needed.
type JobFunc func()

// Run calls j()
func (j JobFunc) Run() {
	j()
}
