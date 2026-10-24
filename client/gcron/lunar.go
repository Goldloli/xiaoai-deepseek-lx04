package gcron

import (
	"fmt"
	"time"
)

// 农历数据结构
type LunarDate struct {
	Year   int    // 农历年
	Month  int    // 农历月 (1-12)
	Day    int    // 农历日 (1-30)
	IsLeap bool   // 是否为闰月
	Str    string // 中文格式字符串
}

// 1900-2100年间的农历数据表（十六进制表示）
var lunarInfo = []int{
	0x04bd8, 0x04ae0, 0x0a570, 0x054d5, 0x0d260, 0x0d950, 0x16554, 0x056a0, 0x09ad0, 0x055d2, //1900-1909
	0x04ae0, 0x0a5b6, 0x0a4d0, 0x0d250, 0x1d255, 0x0b540, 0x0d6a0, 0x0ada2, 0x095b0, 0x14977, //1910-1919
	0x04970, 0x0a4b0, 0x0b4b5, 0x06a50, 0x06d40, 0x1ab54, 0x02b60, 0x09570, 0x052f2, 0x04970, //1920-1929
	0x06566, 0x0d4a0, 0x0ea50, 0x16a95, 0x05ad0, 0x02b60, 0x186e3, 0x092e0, 0x1c8d7, 0x0c950, //1930-1939
	0x0d4a0, 0x1d8a6, 0x0b550, 0x056a0, 0x1a5b4, 0x025d0, 0x092d0, 0x0d2b2, 0x0a950, 0x0b557, //1940-1949
	0x06ca0, 0x0b550, 0x15355, 0x04da0, 0x0a5b0, 0x14573, 0x052b0, 0x0a9a8, 0x0e950, 0x06aa0, //1950-1959
	0x0aea6, 0x0ab50, 0x04b60, 0x0aae4, 0x0a570, 0x05260, 0x0f263, 0x0d950, 0x05b57, 0x056a0, //1960-1969
	0x096d0, 0x04dd5, 0x04ad0, 0x0a4d0, 0x0d4d4, 0x0d250, 0x0d558, 0x0b540, 0x0b6a0, 0x195a6, //1970-1979
	0x095b0, 0x049b0, 0x0a974, 0x0a4b0, 0x0b27a, 0x06a50, 0x06d40, 0x0af46, 0x0ab60, 0x09570, //1980-1989
	0x04af5, 0x04970, 0x064b0, 0x074a3, 0x0ea50, 0x06b58, 0x05ac0, 0x0ab60, 0x096d5, 0x092e0, //1990-1999
	0x0c960, 0x0d954, 0x0d4a0, 0x0da50, 0x07552, 0x056a0, 0x0abb7, 0x025d0, 0x092d0, 0x0cab5, //2000-2009
	0x0a950, 0x0b4a0, 0x0baa4, 0x0ad50, 0x055d9, 0x04ba0, 0x0a5b0, 0x15176, 0x052b0, 0x0a930, //2010-2019
	0x07954, 0x06aa0, 0x0ad50, 0x05b52, 0x04b60, 0x0a6e6, 0x0a4e0, 0x0d260, 0x0ea65, 0x0d530, //2020-2029
	0x05aa0, 0x076a3, 0x096d0, 0x04afb, 0x04ad0, 0x0a4d0, 0x1d0b6, 0x0d250, 0x0d520, 0x0dd45, //2030-2039
	0x0b5a0, 0x056d0, 0x055b2, 0x049b0, 0x0a577, 0x0a4b0, 0x0aa50, 0x1b255, 0x06d20, 0x0ada0, //2040-2049
	0x14b63, 0x09370, 0x049f8, 0x04970, 0x064b0, 0x168a6, 0x0ea50, 0x06aa0, 0x1a6c4, 0x0aae0, //2050-2059
	0x092e0, 0x0d2e3, 0x0c960, 0x0d557, 0x0d4a0, 0x0da50, 0x05d55, 0x056a0, 0x0a6d0, 0x055d4, //2060-2069
	0x052d0, 0x0a9b8, 0x0a950, 0x0b4a0, 0x0b6a6, 0x0ad50, 0x055a0, 0x0aba4, 0x0a5b0, 0x052b0, //2070-2079
	0x0b273, 0x06930, 0x07337, 0x06aa0, 0x0ad50, 0x14b55, 0x04b60, 0x0a570, 0x054e4, 0x0d160, //2080-2089
	0x0e968, 0x0d520, 0x0daa0, 0x16aa6, 0x056d0, 0x04ae0, 0x0a9d4, 0x0a2d0, 0x0d150, 0x0f252, //2090-2099
}

// 公历转农历函数
func SolarToLunar(solarDate time.Time) *LunarDate {
	baseDate := time.Date(1900, 1, 31, 0, 0, 0, 0, time.Local)
	offsetDays := int(solarDate.Sub(baseDate).Hours() / 24)

	// 计算农历年份
	lunarYear := 1900
	for offsetDays > 0 {
		daysInYear := lunarYearDays(lunarYear)
		if offsetDays < daysInYear {
			break
		}
		offsetDays -= daysInYear
		lunarYear++
	}

	// 计算农历月份和日
	leapMonth := leapMonth(lunarYear)
	daysInMonth := 0
	lunarMonth := 1
	isLeap := false

	for ; lunarMonth <= 12; lunarMonth++ {
		if leapMonth > 0 && lunarMonth == (leapMonth+1) && !isLeap {
			// 处理闰月
			daysInMonth = leapDays(lunarYear)
			isLeap = true
			lunarMonth-- // 补偿月份自增
		} else {
			daysInMonth = monthDays(lunarYear, lunarMonth)
			isLeap = false
		}

		if offsetDays < daysInMonth {
			break
		}
		offsetDays -= daysInMonth
	}

	// 生成中文日期字符串
	monthStr := []string{"正", "二", "三", "四", "五", "六", "七", "八", "九", "十", "冬", "腊"}[lunarMonth-1]
	if isLeap {
		monthStr = "闰" + monthStr
	}
	dayStr := []string{"初一", "初二", "初三", "初四", "初五", "初六", "初七", "初八", "初九", "初十",
		"十一", "十二", "十三", "十四", "十五", "十六", "十七", "十八", "十九", "二十",
		"廿一", "廿二", "廿三", "廿四", "廿五", "廿六", "廿七", "廿八", "廿九", "三十"}[offsetDays]

	var lunarStr string
	if lunarYear == solarDate.Year() {
		lunarStr = fmt.Sprintf("%s月%s", monthStr, dayStr)
	} else {
		lunarStr = fmt.Sprintf("农历%d年·%s月%s", lunarYear, monthStr, dayStr)
	}
	return &LunarDate{
		Year:   lunarYear,
		Month:  lunarMonth,
		Day:    offsetDays + 1,
		IsLeap: isLeap,
		Str:    lunarStr, //fmt.Sprintf("农历%d年%s月%s", lunarYear, monthStr, dayStr),
	}
}

func LunarMonthStr(lunarMonth int) string {
	if lunarMonth > 12 {
		return ""
	}
	return []string{"正", "二", "三", "四", "五", "六", "七", "八", "九", "十", "冬", "腊"}[lunarMonth-1]
}
func LunarDayStr(lunarDay int) string {
	if lunarDay > 30 {
		return ""
	}
	return []string{"初一", "初二", "初三", "初四", "初五", "初六", "初七", "初八", "初九", "初十",
		"十一", "十二", "十三", "十四", "十五", "十六", "十七", "十八", "十九", "二十",
		"廿一", "廿二", "廿三", "廿四", "廿五", "廿六", "廿七", "廿八", "廿九", "三十"}[lunarDay-1]
}

// 获取农历年总天数
func lunarYearDays(year int) int {
	days := 348 // 12个月的基础天数
	for i := 0x8000; i > 0x8; i >>= 1 {
		if (lunarInfo[year-1900] & i) != 0 {
			days += 1
		}
	}
	return days + leapDays(year)
}

// 获取闰月天数
func leapDays(year int) int {
	if leapMonth(year) != 0 {
		if (lunarInfo[year-1900] & 0x10000) != 0 {
			return 30
		}
		return 29
	}
	return 0
}

// 获取月份天数
func monthDays(year, month int) int {
	if (lunarInfo[year-1900] & (0x10000 >> uint(month))) != 0 {
		return 30
	}
	return 29
}

// 获取闰月月份
func leapMonth(year int) int {
	return lunarInfo[year-1900] & 0xf
}

func IsLeapMonth(year, month int) bool {
	leap := leapMonth(year)
	return (month == leap)
}

// 农历转公历函数(默认均非闰月)
func LunarBuildToSolar(year, month, day int, basetime time.Time) time.Time {
	// 基础日期：1900年正月初一（公历1900-01-31）
	hour, min, sec := basetime.Clock()
	baseDate := time.Date(1900, 1, 31, hour, min, sec, 0, basetime.Location())
	totalDays := 0

	// 计算目标年份前的总天数（1900年到year-1年）
	for y := 1900; y < year; y++ {
		totalDays += lunarYearDays(y) // 累加每年天数
	}

	// 计算目标月份前的天数
	leapMonthVal := leapMonth(year)
	for m := 1; m < month; m++ {
		// 处理闰月：当前月是闰月且未计算过
		if leapMonthVal == m {
			totalDays += leapDays(year) // 添加闰月天数
		}
		totalDays += monthDays(year, m) // 添加普通月天数
	}

	// 添加当月天数（day从1开始）
	totalDays += day - 1

	// 转换为公历日期
	return baseDate.AddDate(0, 0, totalDays)
}

// 农历转公历函数
func LunarToSolar(lunar *LunarDate, basetime time.Time) time.Time {
	// 基础日期：1900年1月31日（对应农历1900年正月初一）
	hour, min, sec := basetime.Clock()
	baseDate := time.Date(1900, 1, 31, hour, min, sec, 0, basetime.Location())
	totalDays := 0

	// 计算目标年份前的总天数
	for year := 1900; year < lunar.Year; year++ {
		totalDays += lunarYearDays(year)
	}

	// 计算目标月份前的天数
	leapMonthVal := leapMonth(lunar.Year)
	for month := 1; month < lunar.Month; month++ {
		// 处理闰月逻辑
		if leapMonthVal > 0 && month == leapMonthVal {
			totalDays += leapDays(lunar.Year) // 添加闰月天数
		}
		totalDays += monthDays(lunar.Year, month)
	}

	// 处理当前闰月状态
	if lunar.IsLeap {
		if leapMonthVal != lunar.Month {
			return time.Time{} // 闰月校验失败
		}
		totalDays += monthDays(lunar.Year, lunar.Month) // 闰月需加上当月的天数
	}

	// 添加当月已过天数 (农历日从1开始)
	totalDays += lunar.Day - 1

	// 转换为公历日期
	return baseDate.AddDate(0, 0, totalDays)
}

// 在原有代码末尾添加以下闰月校验函数
func validateLeapMonth(year, month int, isLeap bool) bool {
	leap := leapMonth(year)
	if isLeap {
		return leap == month
	}
	return true
}

/*
一个农历平年约为354或355天，而一个农历闰年可长达383至385天。
虽然春节在公历上的日期年年不同，但它永远不会跑出1月和2月这两个月份。
一个实用的建议：
如果您需要在程序或逻辑中进行判断，一个安全的做法是直接将天数设置为覆盖最长间隔，即加52天（因为最长间隔是51天，加52天可以确保万无一失）。这样虽然在某些年份会“浪费”几天，但能保证在所有情况下都成功跨入下一个农历年。

*/
