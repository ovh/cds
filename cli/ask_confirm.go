package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// AskForConfirmation ask for yes/no confirmation on command line
func AskForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

// MultiChoice for multiple choices question. It returns the selected option
func MultiChoice(s string, opts ...string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(s)
	for i, o := range opts {
		fmt.Printf("\t%s [%d]\n", o, (i + 1))
	}

	for {
		fmt.Printf("Your choice [1-%d]: ", len(opts))

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		for i, o := range opts {
			trimmedResponse := strings.TrimSpace(response)
			n, _ := strconv.Atoi(trimmedResponse)
			if n == i+1 {
				return o
			}
		}
		fmt.Println("wrong choice")
	}
}
