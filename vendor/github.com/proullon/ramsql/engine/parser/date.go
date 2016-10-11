package parser

import (
	"fmt"
	"time"
)

// ParseDate intends to parse all SQL date format
func ParseDate(data string) (*time.Time, error) {
	const long = "2006-01-02 15:04:05.999999999 -0700 MST"
	const short = "2006-Jan-02"

	t, err := time.Parse(long, data)
	if err == nil {
		return &t, nil
	}

	t, err = time.Parse(time.RFC3339, data)
	if err == nil {
		return &t, nil
	}

	t, err = time.Parse(short, data)
	if err == nil {
		return &t, nil
	}

	return nil, fmt.Errorf("not a date")
}
