package gcron

/*
https://github.com/NateScarlet/holiday-cn 中国法定节假日数据
https://raw.githubusercontent.com/NateScarlet/holiday-cn/master/{year}.json
格式：
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "https://raw.githubusercontent.com/NateScarlet/holiday-cn/master/schema.json",
    "type": "object",
    "properties": {
        "year": {
            "type": "number",
            "description": "年份"
        },
        "papers": {
            "type": "array",
            "items": {
                "type": "string"
            },
            "description": "所用国务院文件网址列表"
        },
        "days": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "name": {
                        "type": "string",
                        "description": "节日名称"
                    },
                    "date": {
                        "type": "string",
                        "description": "ISO 8601 日期"
                    },
                    "isOffDay": {
                        "type": "boolean",
                        "description": "是否为休息日"
                    }
                },
                "required": [
                    "name",
                    "date",
                    "isOffDay"
                ]
            }
        }
    },
    "required": [
        "year",
        "papers",
        "days"
    ]
}

2026示例数据：https://raw.githubusercontent.com/NateScarlet/holiday-cn/master/2026.json
{
    "$schema": "https://raw.githubusercontent.com/NateScarlet/holiday-cn/master/schema.json",
    "$id": "https://raw.githubusercontent.com/NateScarlet/holiday-cn/master/2026.json",
    "year": 2026,
    "papers": [
        "https://www.gov.cn/zhengce/zhengceku/202511/content_7047091.htm"
    ],
    "days": [
        {
            "name": "元旦",
            "date": "2026-01-01",
            "isOffDay": true
        },
        {
            "name": "元旦",
            "date": "2026-01-02",
            "isOffDay": true
        },
        {
            "name": "元旦",
            "date": "2026-01-03",
            "isOffDay": true
        },
        {
            "name": "元旦",
            "date": "2026-01-04",
            "isOffDay": false
        },
        {
            "name": "春节",
            "date": "2026-02-14",
            "isOffDay": false
        },
        {
            "name": "春节",
            "date": "2026-02-15",
            "isOffDay": true
        },
        {
            "name": "春节",
            "date": "2026-02-16",
            "isOffDay": true
        },
        {
            "name": "春节",
            "date": "2026-02-17",
            "isOffDay": true
        },
        {
            "name": "春节",
            "date": "2026-02-18",
            "isOffDay": true
        },
        {
            "name": "春节",
            "date": "2026-02-19",
            "isOffDay": true
        },
        {
            "name": "春节",
            "date": "2026-02-20",
            "isOffDay": true
        },
        {
            "name": "春节",
            "date": "2026-02-21",
            "isOffDay": true
        },
        {
            "name": "春节",
            "date": "2026-02-22",
            "isOffDay": true
        },
        {
            "name": "春节",
            "date": "2026-02-23",
            "isOffDay": true
        },
        {
            "name": "春节",
            "date": "2026-02-28",
            "isOffDay": false
        },
        {
            "name": "清明节",
            "date": "2026-04-04",
            "isOffDay": true
        },
        {
            "name": "清明节",
            "date": "2026-04-05",
            "isOffDay": true
        },
        {
            "name": "清明节",
            "date": "2026-04-06",
            "isOffDay": true
        },
        {
            "name": "劳动节",
            "date": "2026-05-01",
            "isOffDay": true
        },
        {
            "name": "劳动节",
            "date": "2026-05-02",
            "isOffDay": true
        },
        {
            "name": "劳动节",
            "date": "2026-05-03",
            "isOffDay": true
        },
        {
            "name": "劳动节",
            "date": "2026-05-04",
            "isOffDay": true
        },
        {
            "name": "劳动节",
            "date": "2026-05-05",
            "isOffDay": true
        },
        {
            "name": "劳动节",
            "date": "2026-05-09",
            "isOffDay": false
        },
        {
            "name": "端午节",
            "date": "2026-06-19",
            "isOffDay": true
        },
        {
            "name": "端午节",
            "date": "2026-06-20",
            "isOffDay": true
        },
        {
            "name": "端午节",
            "date": "2026-06-21",
            "isOffDay": true
        },
        {
            "name": "国庆节",
            "date": "2026-09-20",
            "isOffDay": false
        },
        {
            "name": "中秋节",
            "date": "2026-09-25",
            "isOffDay": true
        },
        {
            "name": "中秋节",
            "date": "2026-09-26",
            "isOffDay": true
        },
        {
            "name": "中秋节",
            "date": "2026-09-27",
            "isOffDay": true
        },
        {
            "name": "国庆节",
            "date": "2026-10-01",
            "isOffDay": true
        },
        {
            "name": "国庆节",
            "date": "2026-10-02",
            "isOffDay": true
        },
        {
            "name": "国庆节",
            "date": "2026-10-03",
            "isOffDay": true
        },
        {
            "name": "国庆节",
            "date": "2026-10-04",
            "isOffDay": true
        },
        {
            "name": "国庆节",
            "date": "2026-10-05",
            "isOffDay": true
        },
        {
            "name": "国庆节",
            "date": "2026-10-06",
            "isOffDay": true
        },
        {
            "name": "国庆节",
            "date": "2026-10-07",
            "isOffDay": true
        },
        {
            "name": "国庆节",
            "date": "2026-10-10",
            "isOffDay": false
        }
    ]
}
*/

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"os"
	"sync"
	"time"
)

// Holiday 单个节假日/调休日期的结构体
type Holiday struct {
	Name     string `json:"name"`
	Date     string `json:"date"`
	IsOffDay bool   `json:"isOffDay"`
}

// HolidayData 全年节假日数据的结构体
type HolidayData struct {
	Year   int       `json:"year"`
	Days   []Holiday `json:"days"`
	Papers []string  `json:"papers"` // 保留字段，兼容JSON结构
}

// holidayCache 缓存已加载的年份节假日数据，key为年份(int)
var holidayCache = make(map[int]map[string]Holiday)
var cacheMutex sync.RWMutex // 读写锁，保证并发安全

// getHolidayData 获取指定年份的节假日数据（优先从缓存，无则下载）
func getHolidayData(year int) (map[string]Holiday, error) {
	// 先尝试从缓存读取（读锁）
	cacheMutex.RLock()
	data, ok := holidayCache[year]
	cacheMutex.RUnlock()
	if ok {
		return data, nil
	}

	// 缓存未命中
	var holidayData HolidayData
	filepath := fmt.Sprintf("%d.json", year)

	// 读取文件内容
	readFromfile := func(filepath string) error {
		file, err := os.Open(filepath)
		if err != nil {
			return err
		}
		defer file.Close()

		content, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}

		// 解析JSON
		if err := json.Unmarshal([]byte(content), &holidayData); err != nil {
			return err
		}
		return nil
	}
	if !IsExist(filepath) || readFromfile(filepath) != nil {
		// 读本地文件失败，下载数据
		url := fmt.Sprintf("https://raw.githubusercontent.com/NateScarlet/holiday-cn/master/%d.json", year)
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("下载%d年节假日数据失败: %w", year, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("下载%d年节假日数据失败，状态码: %d", year, resp.StatusCode)
		}

		//完整读取Body到内存
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("下载%d年节假日数据失败: %w", year, err)
		}
		// 解析JSON数据
		if err := json.Unmarshal(bodyBytes, &holidayData); err != nil {
			return nil, fmt.Errorf("解析%d年节假日数据失败: %w", year, err)
		}
		//将原始字节写入文件
		os.WriteFile(filepath, bodyBytes, 0644)
	}

	// 构建日期映射（key为"2026-01-01"格式的日期字符串）
	dateMap := make(map[string]Holiday, len(holidayData.Days))
	for _, day := range holidayData.Days {
		dateMap[day.Date] = day
	}

	// 写入缓存（写锁）
	cacheMutex.Lock()
	holidayCache[year] = dateMap
	cacheMutex.Unlock()

	return dateMap, nil
}

// IsHoliday 判断指定日期是否为节假日/休息日
// 返回true表示是休息日（包括法定假日、调休后的休息日、周末）
func IsHoliday(t time.Time) bool {
	// 格式化日期为 "YYYY-MM-DD"
	dateStr := t.Format("2006-01-02")
	year := t.Year()

	// 获取该年的节假日数据
	holidayMap, err := getHolidayData(year)
	if err == nil {
		// 检查是否在节假日列表中
		if holiday, ok := holidayMap[dateStr]; ok {
			return holiday.IsOffDay
		}
	}

	// 未在列表中，按自然周规则判断
	weekday := t.Weekday()
	// 周六(6)、周日(0)为休息日
	return weekday == time.Saturday || weekday == time.Sunday
}

// IsWorkday 判断指定日期是否为工作日
// 返回true表示是工作日（包括调休后的工作日、正常工作日）
func IsWorkday(t time.Time) bool {
	// 格式化日期为 "YYYY-MM-DD"
	dateStr := t.Format("2006-01-02")
	year := t.Year()

	// 获取该年的节假日数据
	holidayMap, err := getHolidayData(year)
	if err == nil {
		// 检查是否在节假日列表中
		if holiday, ok := holidayMap[dateStr]; ok {
			return !holiday.IsOffDay
		}
	}

	// 未在列表中，按自然周规则判断
	weekday := t.Weekday()
	// 周一(1)到周五(5)为工作日
	return weekday >= time.Monday && weekday <= time.Friday
}
