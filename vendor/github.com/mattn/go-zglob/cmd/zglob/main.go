package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mattn/go-zglob"
)

func main() {
	var d bool
	flag.BoolVar(&d, "d", false, "with directory")
	flag.Parse()
	for _, arg := range os.Args[1:] {
		matches, err := zglob.Glob(arg)
		if err != nil {
			continue
		}
		for _, m := range matches {
			if !d {
				if fi, err := os.Stat(m); err == nil && fi.Mode().IsDir() {
					continue
				}
			}
			fmt.Println(m)
		}
	}
}
