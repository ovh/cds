package main

import (
	"os"

	"github.com/ovh/cds/sdk/plugin"
)

func main() {
	app := initCli(func() {
		plugin.Serve(&KafkaPlugin{})
	})
	app.Run(os.Args)
}
