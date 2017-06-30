package cli

import (
	"bufio"
	"os"
	"strings"

	"github.com/ovh/cds/sdk"
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
			sdk.Exit("Error: cannot read from stdin (%s)\n", err)
		}
		all += string(line)
	}

	return strings.Replace(all, "\n", "", -1)
}
