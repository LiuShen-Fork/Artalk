package handler

import (
	"sort"
	"time"

	"gorm.io/gorm"
)

// dayCount 是某个自然日的计数，Date 为 YYYY-MM-DD
type dayCount struct {
	Date  string
	Count int64
}

// countByLocalDay 统计 query 命中的记录自 since 起每天的数量。
//
// 为什么不直接用 SQL 的 DATE() / strftime() 分组：数据库的日期函数跟随的是
// 会话或进程时区（SQLite 的 'localtime' 修饰符跟随进程 TZ 环境变量，Postgres
// 跟随会话 TimeZone），而 Artalk 的时区来自配置文件。部署在 UTC 服务器上时，
// 两者会相差几个小时，跨零点前后的记录就会被算到前一天。这里只取出 created_at，
// 在 Go 侧按 time.Local（启动时已设为配置的时区）归日，结果与数据库无关。
func countByLocalDay(query *gorm.DB, since time.Time) ([]dayCount, error) {
	var times []time.Time
	if err := query.Where("created_at >= ?", since).Pluck("created_at", &times).Error; err != nil {
		return nil, err
	}

	counts := make(map[string]int64, len(times))
	for _, at := range times {
		counts[at.In(time.Local).Format("2006-01-02")]++
	}

	days := make([]dayCount, 0, len(counts))
	for date, count := range counts {
		days = append(days, dayCount{Date: date, Count: count})
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })
	return days, nil
}
