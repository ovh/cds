package main

import (
	"github.com/ovh/cds/sdk/plugin"
)

// Plugin entrypoint
func main() {
	plugin.Main(&Plugin{})
}
