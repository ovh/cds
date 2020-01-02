package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ovh/cds/sdk/cdsclient"
)

var host, token, username string

func init() {
	flag.StringVar(&host, "api", "http://localhost:8081", "CDS API URL, ex: http://localhost:8081")
	flag.StringVar(&token, "token", "", "CDS Token")
	flag.StringVar(&username, "username", "", "CDS Username")
}

func main() {
	flag.Parse()
	cfg := cdsclient.Config{
		Host:         host,
		SessionToken: token,
		User:         username,
	}
	client := cdsclient.New(cfg)

	// go on https://godoc.org/github.com/ovh/cds/sdk/cdsclient to
	// see all available funcs
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	workers, err := client.WorkerList(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while getting workers:%s", err)
		os.Exit(1)
	}

	if len(workers) == 0 {
		fmt.Println("> No worker")
	} else {
		fmt.Println("Current Workers:")
		for _, w := range workers {
			fmt.Printf("> %s\n", w.Name)
		}
	}

}
