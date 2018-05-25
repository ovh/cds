package sdk

import (
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
