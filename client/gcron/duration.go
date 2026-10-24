package gcron

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// extends time.Duration

const (
	//Day has 24 hours
	Day time.Duration = time.Hour * 24
	//Week has 7 days
	Week time.Duration = Day * 7
)

var (
	ErrTimesEmpty = errors.New("空字符串")
	ErrTimesValid = errors.New("次数")
)

var unitMap = map[string]time.Duration{
	"s": time.Second, "sec": time.Second, "second": time.Second, "秒": time.Second,
	"m": time.Minute, "min": time.Minute, "minute": time.Minute, "分钟": time.Minute, "分": time.Minute,
	"h": time.Hour, "hr": time.Hour, "hour": time.Hour, "小时": time.Hour, "时": time.Hour,
	"d": Day, "day": Day, "天": Day,
	"w": Week, "week": Week, "周": Week,
}

func ParseDuration(s string) (time.Duration, error) {
	var total time.Duration
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrTimesEmpty
	}

	//次数：t,times,次 (无单位也默认为次数)
	s = strings.ReplaceAll(s, "次", "")
	s = strings.ReplaceAll(s, "times", "")
	s = strings.ReplaceAll(s, "t", "")
	s = strings.TrimSpace(s)
	n, err := strconv.Atoi(s)
	if err == nil {
		return time.Duration(n), ErrTimesValid
	}

	for s != "" {
		// 提取数字部分
		i := 0
		for ; i < len(s); i++ {
			if !unicode.IsDigit(rune(s[i])) && s[i] != '.' {
				break
			}
		}
		if i == 0 {
			return 0, fmt.Errorf("无效格式: %q", s)
		}

		// 解析数值
		numStr := s[:i]
		value, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, fmt.Errorf("无效数字: %q", numStr)
		}

		// 提取并匹配单位
		s = strings.TrimSpace(s[i:])
		if s == "" {
			return 0, errors.New("缺少单位")
		}

		j := 0
		for ; j < len(s); j++ {
			if unicode.IsSpace(rune(s[j])) || unicode.IsDigit(rune(s[j])) {
				break
			}
		}

		unitStr := strings.ToLower(s[:j])
		s = strings.TrimSpace(s[j:])
		matched := false

		for suffix, unit := range unitMap {
			if strings.HasPrefix(unitStr, suffix) {
				total += time.Duration(value * float64(unit))
				matched = true
				break
			}
		}

		if !matched {
			return 0, fmt.Errorf("未知单位: %q", unitStr)
		}
	}
	return total, nil
}
