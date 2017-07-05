package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

//ReadLine prompts input from the user delimited by a new line
func ReadLine() string {
	var all string
	var line []byte
	var err error

	hasMoreInLine := true
	bio := bufio.NewReader(os.Stdin)

	for hasMoreInLine {
		line, hasMoreInLine, err = bio.ReadLine()
		if err != nil {
			fmt.Println("Error: cannot read from stdin", err)
			os.Exit(1)
		}
		all += string(line)
	}

	return strings.Replace(all, "\n", "", -1)
}
