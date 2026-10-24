package gcron

import (
	"errors"
	"time"
	//"github.com/roylee0704/gron/xtime"
)

// Schedule is the interface that wraps the basic Next method.
//
// Next deduces next occurring time based on t and underlying states.
type Schedule interface {
	Next(t time.Time) time.Time
}

// AtSchedule extends Schedule by enabling periodic-interval & time-specific setup
type AtSchedule interface {
	At(t string) Schedule
	Schedule
}

// Every returns a Schedule reoccurs every period p, p must be at least
// time.Second.
func Every(p time.Duration) AtSchedule {

	if p < time.Second {
		p = time.Second
	}

	p = p - time.Duration(p.Nanoseconds())%time.Second // truncates up to seconds

	return &periodicSchedule{
		period: p,
	}
}

//下边的periodicSchedule 以及 atSchedule 就是 Schedule 接口的具体实现。
//我们也完全可以不用 gron.Every，而是自己写一套新的 Schedule 实现。只要实现 Next(p time.Duration) time.Time 即可。

type periodicSchedule struct {
	period time.Duration
}

// Next adds time t to underlying period, truncates up to unit of seconds.
func (ps periodicSchedule) Next(t time.Time) time.Time {
	return t.Truncate(time.Second).Add(ps.period)
	//把秒以下的截掉之后，直接 Add(period)，把周期加到当前的 time.Time 上，返回新的时间点。
}

// At returns a schedule which reoccurs every period p, at time t(hh:ss).
//
// Note: At panics when period p is less than xtime.Day, and error hh:ss format.
/*
对 At 能力的支持。我们来关注下 func (ps periodicSchedule) At(t string) Schedule 这个方法
· 若周期连 1 天都不到，不支持 At 能力，因为 At 本质是在选定的一天内，指定小时，分钟，作为辅助。连一天都不到的周期，是要精准处理的；
· 将用户输入的形如 "23:59" 时间字符串解析出来【小时】和【分钟】；
· 构建出一个 atSchedule 对象，包含了【周期时长】，【小时】，【分钟】。*/
func (ps periodicSchedule) At(t string) Schedule {
	if ps.period < Day {
		panic("period must be at least in days")
	}

	// parse t naively
	h, m, err := parse(t)

	if err != nil {
		panic(err.Error())
	}

	return &atSchedule{
		period: ps.period,
		hh:     h,
		mm:     m,
	}
}

// parse naively tokenises hours and minutes.
//
// returns error when input format was incorrect.
func parse(hhmm string) (hh int, mm int, err error) {

	hh = int(hhmm[0]-'0')*10 + int(hhmm[1]-'0')
	mm = int(hhmm[3]-'0')*10 + int(hhmm[4]-'0')

	if hh < 0 || hh > 24 {
		hh, mm = 0, 0
		err = errors.New("invalid hh format")
	}
	if mm < 0 || mm > 59 {
		hh, mm = 0, 0
		err = errors.New("invalid mm format")
	}

	return
}

type atSchedule struct {
	period time.Duration
	hh     int
	mm     int
}

// reset returns new Date based on time instant t, and reconfigure its hh:ss
// according to atSchedule's hh:ss.
func (as atSchedule) reset(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), as.hh, as.mm, 0, 0, time.UTC)
	//根据原有 time.Time 的年，月，日，以及用户输入的 At 中的小时，分钟，来构建出来一个 time.Time 作为新的时间点。
}

// Next returns **next** time.
// if t passed its supposed schedule: reset(t), returns reset(t) + period,
// else returns reset(t).
func (as atSchedule) Next(t time.Time) time.Time {
	next := as.reset(t)
	if t.After(next) {
		return next.Add(as.period)
	}
	return next
}

/*
在调用 Next 方法时，先做 reset，根据原有 time.Time 的年，月，日，以及用户输入的 At 中的小时，分钟，来构建出来一个 time.Time 作为新的时间点。
此后判断是在哪个周期，如果当前周期已经过了，那就按照下个周期的时间点返回。
到这里，一切就都清楚了，如果我们不用 At 能力，直接 gron.Every(xxx)，那么直接就会调用
t.Truncate(time.Second).Add(ps.period)
拿到一个新的时间点返回。
而如果我们要用 At 能力，指定当天的小时，分钟。那就会走到 periodicSchedule.At 这里，解析出【小时】和【分钟】，最后走 Next 返回 reset 之后的时间点。
这个和 gron.Every 方法返回的 AtSchedule 接口其实是完全对应的：
// AtSchedule extends Schedule by enabling periodic-interval & time-specific setuptype AtSchedule interface {
  At(t string) Schedule
  Schedule
直接就有一个 Schedule 可以用，但如果你想针对天级以上的 duration 指定时间，也可以走 At 方法，也会返回一个 Schedule 供我们使用。
*/
