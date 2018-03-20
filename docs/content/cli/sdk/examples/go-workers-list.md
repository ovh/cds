+++
title = "Workers List"
+++


### List workers

Create a file `main.go` with this content:

```go

package main

import (
	"flag"
	"fmt"
	"os"

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
		Host:  host,
		Token: token,
		User:  username,
	}
	client := cdsclient.New(cfg)

	workers, err := client.WorkerList()
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

```

## Build & run it

```bash
go run main.go --username admin --token xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --api http://localhost:8081
```