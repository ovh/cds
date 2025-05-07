package sdk

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// IsInArray checks if the element is in the array
func IsInArray[T comparable](elt T, array []T) bool {
	for _, item := range array {
		if item == elt {
			return true
		}
	}
	return false
}

// IsInInt64Array checks if the element is in the array (int64)
func IsInInt64Array(elt int64, array []int64) bool {
	for _, item := range array {
		if item == elt {
			return true
		}
	}
	return false
}

// RandomString generate random string containing a-zA-Z0-9
func RandomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// DeleteEmptyValueFromArray deletes empty value from an array of string
func DeleteEmptyValueFromArray(array []string) []string {
	out := make([]string, 0, len(array))
	for _, str := range array {
		if str != "" {
			out = append(out, str)
		}
	}
	return out
}

// DeleteFromArray deletes value from an array of string
func DeleteFromArray(array []string, el string) []string {
	out := make([]string, 0, len(array))
	for _, str := range array {
		if str != el {
			out = append(out, str)
		}
	}
	return out
}

// DeleteFromArray deletes value from an array of int64
func DeleteFromInt64Array(array []int64, el int64) []int64 {
	out := make([]int64, 0, len(array))
	for _, str := range array {
		if str != el {
			out = append(out, str)
		}
	}
	return out
}

// IntMapToSlice converts a map struct to a slice for int64 keys
func IntMapToSlice(m map[int64]struct{}) []int64 {
	slice := make([]int64, 0, len(m))
	for i := range m {
		slice = append(slice, i)
	}
	return slice
}

func StringFirstN(s string, i int) string {
	if len(s) <= i {
		return s
	}
	return s[:i]
}

type ReqNotHostMatcher struct {
	NotHost string
}

func (m ReqNotHostMatcher) Matches(x interface{}) bool {
	switch i := x.(type) {
	case *http.Request:
		return i.URL.Host != m.NotHost
	default:
		return false
	}
}

func (m ReqNotHostMatcher) String() string {
	return fmt.Sprintf("Not Host is %q", m.NotHost)
}

type ReqHostMatcher struct {
	Host string
}

func (m ReqHostMatcher) Matches(x interface{}) bool {
	switch i := x.(type) {
	case *http.Request:
		return i.URL.Host == m.Host
	default:
		return false
	}
}

func (m ReqHostMatcher) String() string {
	return fmt.Sprintf("Host is %q", m.Host)
}

type ReqMatcher struct {
	Method  string
	URLPath string
}

func (m ReqMatcher) Matches(x interface{}) bool {
	switch i := x.(type) {
	case *http.Request:
		return i.URL.Path == m.URLPath && m.Method == i.Method
	default:
		return false
	}
}

func (m ReqMatcher) String() string {
	return fmt.Sprintf("Method is %q, URL Path is %q", m.Method, m.URLPath)
}

func Unique[T comparable](s []T) []T {
	inResult := make(map[T]bool)
	var result []T
	for _, str := range s {
		if _, ok := inResult[str]; !ok {
			inResult[str] = true
			result = append(result, str)
		}
	}
	return result
}
