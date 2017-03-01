// Package now is a time toolkit for golang.
//
// More details README here: https://github.com/jinzhu/now
//
//  import "github.com/jinzhu/now"
//
//  now.BeginningOfMinute() // 2013-11-18 17:51:00 Mon
//  now.BeginningOfDay()    // 2013-11-18 00:00:00 Mon
//  now.EndOfDay()          // 2013-11-18 23:59:59.999999999 Mon
package now

import "time"

var FirstDayMonday bool
var TimeFormats = []string{"1/2/2006", "1/2/2006 15:4:5", "2006-1-2 15:4:5", "2006-1-2 15:4", "2006-1-2", "1-2", "15:4:5", "15:4", "15", "15:4:5 Jan 2, 2006 MST", "2006-01-02 15:04:05.999999999 -0700 MST"}

type Now struct {
	time.Time
}

func New(t time.Time) *Now {
	return &Now{t}
}

func BeginningOfMinute() time.Time {
	return New(time.Now()).BeginningOfMinute()
}

func BeginningOfHour() time.Time {
	return New(time.Now()).BeginningOfHour()
}

func BeginningOfDay() time.Time {
	return New(time.Now()).BeginningOfDay()
}

func BeginningOfWeek() time.Time {
	return New(time.Now()).BeginningOfWeek()
}

func BeginningOfMonth() time.Time {
	return New(time.Now()).BeginningOfMonth()
}

func BeginningOfQuarter() time.Time {
	return New(time.Now()).BeginningOfQuarter()
}

func BeginningOfYear() time.Time {
	return New(time.Now()).BeginningOfYear()
}

func EndOfMinute() time.Time {
	return New(time.Now()).EndOfMinute()
}

func EndOfHour() time.Time {
	return New(time.Now()).EndOfHour()
}

func EndOfDay() time.Time {
	return New(time.Now()).EndOfDay()
}

func EndOfWeek() time.Time {
	return New(time.Now()).EndOfWeek()
}

func EndOfMonth() time.Time {
	return New(time.Now()).EndOfMonth()
}

func EndOfQuarter() time.Time {
	return New(time.Now()).EndOfQuarter()
}

func EndOfYear() time.Time {
	return New(time.Now()).EndOfYear()
}

func Monday() time.Time {
	return New(time.Now()).Monday()
}

func Sunday() time.Time {
	return New(time.Now()).Sunday()
}

func EndOfSunday() time.Time {
	return New(time.Now()).EndOfSunday()
}

func Parse(strs ...string) (time.Time, error) {
	return New(time.Now()).Parse(strs...)
}

func MustParse(strs ...string) time.Time {
	return New(time.Now()).MustParse(strs...)
}

func Between(time1, time2 string) bool {
	return New(time.Now()).Between(time1, time2)
}
