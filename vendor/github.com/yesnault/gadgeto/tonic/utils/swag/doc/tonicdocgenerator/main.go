package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/loopfz/gadgeto/tonic/utils/swag/doc"
)

/*
	Run this script to generate swagger schema + sdks
*/

var directory = flag.String("directory", "", "root directory from which godoc will be generated")

func main() {
	flag.Parse()

	if directory == nil {
		fmt.Println("Missing directory param")
		return
	}

	godoc := doc.GenerateDoc(*directory)
	b, err := json.MarshalIndent(godoc, "", "    ")
	if err != nil {
		panic(err)
	}
	godocStr := strings.Replace(string(b), "`", "'", -1)
	fmt.Println(godocStr)

}
