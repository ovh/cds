---
title: "SDK Golang"
card: 
  name: rest-sdk
---

## How to use it?

You have to initialize a cdsclient:

```go
client := cdsclient.New(cdsclient.Config{
	Host:    host,
})
res, err := client.AuthConsumerSignin(sdk.ConsumerBuiltin, sdk.AuthConsumerSigninRequest{
	"token": "<signin-token-value>",
})
client = cdsclient.New(cdsclient.Config{
	Host:  host,
	SessionToken: res.Token,
})
```

and then, you can use it:

```go

// list workers
workers, err := client.WorkerList()

// list users
users, err := client.UserList()

// list workflow runs
runs, err := client.WorkflowRunList(...)

```

Go on https://godoc.org/github.com/ovh/cds/sdk/cdsclient to see all available funcs.
	

## Example

+ Create a file `main.go` with this content:

```go

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk"
)

var host, token string

func init() {
	flag.StringVar(&host, "api", "http://localhost:8081", "CDS API URL, ex: http://localhost:8081")
	flag.StringVar(&token, "token", "", "CDS signin token")
}

func main() {
	flag.Parse()

	client := cdsclient.New(cdsclient.Config{
		Host:    host,
	})
	res, err := client.AuthConsumerSignin(sdk.ConsumerBuiltin, sdk.AuthConsumerSigninRequest{
		"token": token,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while signin: %v", err)
		os.Exit(1)
	}
	client = cdsclient.New(cdsclient.Config{
		Host:  host,
		SessionToken: res.Token,
	})

	workers, err := client.WorkerList()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while getting workers: %v", err)
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

+ Build & run it: 

```bash
go run main.go --token xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --api http://localhost:8081
```
