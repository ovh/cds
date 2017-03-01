package main

import (
	"fmt"
	"os"

	"github.com/ovh/cds/sdk/plugin"
)

// Plugin entrypoint
func main() {
	if len(os.Args) == 2 && os.Args[1] == "info" {
		plugin := &Plugin{}
		params := plugin.Parameters()

		fmt.Printf(" - Name:\t%s\n", plugin.Name())
		fmt.Printf(" - Author:\t%s\n", plugin.Author())
		fmt.Printf(" - Description:\t%s\n", plugin.Description())
		fmt.Println(" - Parameters:")
		for _, n := range params.Names() {
			fmt.Printf("\t - %s: %s\n", n, params.GetDescription(n))
		}
		return
	}

	plugin.Serve(&Plugin{})
}
