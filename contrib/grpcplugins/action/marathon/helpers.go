package main

import "strings"

//Escape characters
func Escape(s string) string {
	s1 := strings.Replace(s, "_", "-", -1)
	s1 = strings.Replace(s1, "/", "-", -1)
	s1 = strings.Replace(s1, ".", "-", -1)
	return s1
}
