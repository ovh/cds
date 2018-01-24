## Now

Now is a time toolkit for golang

#### Why the project named `Now`?

```go
now.BeginningOfDay()
```
`now` is quite readable, aha?

#### But `now` is so common I can't search the project with my favorite search engine

* Star it in github [https://github.com/jinzhu/now](https://github.com/jinzhu/now)
* Documentation at [godoc.org](https://godoc.org/github.com/jinzhu/now)

## Install

```
go get -u github.com/jinzhu/now
```

### Usage

```go
import "github.com/jinzhu/now"

time.Now() // 2013-11-18 17:51:49.123456789 Mon

now.BeginningOfMinute()   // 2013-11-18 17:51:00 Mon
now.BeginningOfHour()     // 2013-11-18 17:00:00 Mon
now.BeginningOfDay()      // 2013-11-18 00:00:00 Mon
now.BeginningOfWeek()     // 2013-11-17 00:00:00 Sun
now.FirstDayMonday = true // Set Monday as first day, default is Sunday
now.BeginningOfWeek()     // 2013-11-18 00:00:00 Mon
now.BeginningOfMonth()    // 2013-11-01 00:00:00 Fri
now.BeginningOfQuarter()  // 2013-10-01 00:00:00 Tue
now.BeginningOfYear()     // 2013-01-01 00:00:00 Tue

now.EndOfMinute()         // 2013-11-18 17:51:59.999999999 Mon
now.EndOfHour()           // 2013-11-18 17:59:59.999999999 Mon
now.EndOfDay()            // 2013-11-18 23:59:59.999999999 Mon
now.EndOfWeek()           // 2013-11-23 23:59:59.999999999 Sat
now.FirstDayMonday = true // Set Monday as first day, default is Sunday
now.EndOfWeek()           // 2013-11-24 23:59:59.999999999 Sun
now.EndOfMonth()          // 2013-11-30 23:59:59.999999999 Sat
now.EndOfQuarter()        // 2013-12-31 23:59:59.999999999 Tue
now.EndOfYear()           // 2013-12-31 23:59:59.999999999 Tue


// Use another time
t := time.Date(2013, 02, 18, 17, 51, 49, 123456789, time.Now().Location())
now.New(t).EndOfMonth()   // 2013-02-28 23:59:59.999999999 Thu


// Don't want be bothered with the First Day setting, Use Monday, Sunday
now.Monday()              // 2013-11-18 00:00:00 Mon
now.Sunday()              // 2013-11-24 00:00:00 Sun (Next Sunday)
now.EndOfSunday()         // 2013-11-24 23:59:59.999999999 Sun (End of next Sunday)

t := time.Date(2013, 11, 24, 17, 51, 49, 123456789, time.Now().Location()) // 2013-11-24 17:51:49.123456789 Sun
now.New(t).Monday()       // 2013-11-18 00:00:00 Sun (Last Monday if today is Sunday)
now.New(t).Sunday()       // 2013-11-24 00:00:00 Sun (Beginning Of Today if today is Sunday)
now.New(t).EndOfSunday()  // 2013-11-24 23:59:59.999999999 Sun (End of Today if today is Sunday)
```

#### Parse String

```go
time.Now() // 2013-11-18 17:51:49.123456789 Mon

// Parse(string) (time.Time, error)
t, err := now.Parse("12:20")            // 2013-11-18 12:20:00, nil
t, err := now.Parse("1999-12-12 12:20") // 1999-12-12 12:20:00, nil
t, err := now.Parse("99:99")            // 2013-11-18 12:20:00, Can't parse string as time: 99:99

// MustParse(string) time.Time
now.MustParse("2013-01-13")             // 2013-01-13 00:00:00
now.MustParse("02-17")                  // 2013-02-17 00:00:00
now.MustParse("2-17")                   // 2013-02-17 00:00:00
now.MustParse("8")                      // 2013-11-18 08:00:00
now.MustParse("2002-10-12 22:14")       // 2002-10-12 22:14:00
now.MustParse("99:99")                  // panic: Can't parse string as time: 99:99
```

Extend `now` to support more formats is quite easy, just update `TimeFormats` variable with `time.Format` like time layout

```go
now.TimeFormats = append(now.TimeFormats, "02 Jan 2006 15:04")
```

Please send me pull requests if you want a format to be supported officially


# Supporting the project

[![http://patreon.com/jinzhu](http://patreon_public_assets.s3.amazonaws.com/sized/becomeAPatronBanner.png)](http://patreon.com/jinzhu)


# Author

**jinzhu**

* <http://github.com/jinzhu>
* <wosmvp@gmail.com>
* <http://twitter.com/zhangjinzhu>

## License

Released under the [MIT License](http://www.opensource.org/licenses/MIT).
