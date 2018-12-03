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
		fmt.Printf("%s [Y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "Y" || response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		} else if response == "" {
			return true
		}
	}
}

// MultiChoice for multiple choices question. It returns the selected option
func MultiChoice(s string, opts ...string) int {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(s)
	if len(opts) == 0 {
		log.Fatal(fmt.Errorf("no choice available"))
	}
	for i, o := range opts {
		fmt.Printf("\t[%d] %s\n", (i + 1), o)
	}

	for {
		if len(opts) > 1 {
			fmt.Printf("Your choice [1-%d]: ", len(opts))
		} else {
			fmt.Printf("Your choice [1]: ")
		}
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		n, _ := strconv.Atoi(strings.TrimSpace(response))
		if 0 < n && n <= len(opts) {
			return n - 1
		}

		fmt.Println("wrong choice")
	}
}

// AskValueChoice ask for a string and returns it.
func AskValueChoice(s string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s", s)

	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	return strings.TrimSpace(response)
}
