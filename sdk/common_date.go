package sdk

import (
	"sync"
	"time"

	"github.com/pkg/errors"
)

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// For the most part, this parsing date functions follows the syntax as specified by RFC 5322 and
// extended by RFC 6532.
// Notable divergences:
// 	* Obsolete address formats are not parsed, including addresses with
// 	  embedded route information.
// 	* The full range of spacing (the CFWS syntax element) is not supported,
// 	  such as breaking addresses across lines.
// 	* No unicode normalization is performed.
// 	* The special characters ()[]:;@\, are allowed to appear unquoted in names.
// Layouts suitable for passing to time.Parse.
// These are tried in order.
var (
	dateLayoutsBuildOnce sync.Once
	dateLayouts          []string
)

func buildDateLayouts() {
	// Generate layouts based on RFC 5322, section 3.3.

	dows := [...]string{"", "Mon, "}   // day-of-week
	days := [...]string{"2", "02"}     // day = 1*2DIGIT
	years := [...]string{"2006", "06"} // year = 4*DIGIT / 2*DIGIT
	seconds := [...]string{":05", ""}  // second
	// "-0700 (MST)" is not in RFC 5322, but is common.
	zones := [...]string{"-0700", "MST", "-0700 (MST)"} // zone = (("+" / "-") 4DIGIT) / "GMT" / ...

	for _, dow := range dows {
		for _, day := range days {
			for _, year := range years {
				for _, second := range seconds {
					for _, zone := range zones {
						s := dow + day + " Jan " + year + " 15:04" + second + " " + zone
						dateLayouts = append(dateLayouts, s)
					}
				}
			}
		}
	}
}

// ParseDateRFC5322 parses an RFC 5322 date string.
func ParseDateRFC5322(date string) (time.Time, error) {
	dateLayoutsBuildOnce.Do(buildDateLayouts)
	for _, layout := range dateLayouts {
		t, err := time.Parse(layout, date)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("date could not be parsed")
}

// FormatDateRFC5322 format an RFC 5322 date string.
func FormatDateRFC5322(date time.Time) string {
	dateLayoutsBuildOnce.Do(buildDateLayouts)
	for _, layout := range dateLayouts {
		t := date.Format(layout)
		return t
	}
	return ""
}
