package sdk

import (
	"math"
	"time"
)

// Round rounds a duration, see original source here: https://play.golang.org/p/WjfKwhhjL5
// round("1h23m45.6789s", time.Second} will returns want: "1h23m46s"
func Round(d, r time.Duration) time.Duration {
	if r <= 0 {
		return d
	}
	neg := d < 0
	if neg {
		d = -d
	}
	if m := d % r; m+m < r {
		d = d - m
	} else {
		d = d + r - m
	}
	if neg {
		return -d
	}
	return d
}

// RoundN round with n digits
func RoundN(d time.Duration, n int) time.Duration {
	if n < 1 {
		return d
	}
	if d >= time.Hour {
		k := digits(d / time.Hour)
		if k >= n {
			return Round(d, time.Hour*time.Duration(math.Pow10(k-n)))
		}
		n -= k
		k = digits(d % time.Hour / time.Minute)
		if k >= n {
			return Round(d, time.Minute*time.Duration(math.Pow10(k-n)))
		}
		return Round(d, time.Duration(float64(100*time.Second)*math.Pow10(k-n)))
	}
	if d >= time.Minute {
		k := digits(d / time.Minute)
		if k >= n {
			return Round(d, time.Minute*time.Duration(math.Pow10(k-n)))
		}
		return Round(d, time.Duration(float64(100*time.Second)*math.Pow10(k-n)))
	}
	if k := digits(d); k > n {
		return Round(d, time.Duration(math.Pow10(k-n)))
	}
	return d
}

func digits(d time.Duration) int {
	if d < 0 {
		d = -d
	}
	i := 1
	for d > 9 {
		d /= 10
		i++
	}
	return i
}
