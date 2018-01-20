---
title: "Golang - A simple main example"
weight: 1
toc: true
prev: "/sdk"
next: "/sdk/golang-full-example"

---


## Usage
```
 cd <directory-containing-main.go>/
 go get -u github.com/ovh/tat
 build && ./mycli-minimal -url=http://url-tat-engine -username=<tatUsername> -password=<tatPassword> /Internal/your/topic your message
```

## File main.go
```go
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ovh/tat"
)

// taturl, username / password of tat engine
var (
	taturl   string
	username string
	password string
)

func main() {
	flag.StringVar(&taturl, "url", "", "URL of Tat Engine")
	flag.StringVar(&username, "username", "", "tat username")
	flag.StringVar(&password, "password", "", "tat password")
	flag.Parse()

	client, err := tat.NewClient(tat.Options{
		URL:      taturl,
		Username: username,
		Password: password,
		Referer:  "mycli-minimal.v0",
	})

	if err != nil {
		fmt.Printf("Error while create new Tat Client: %s\n", err)
		os.Exit(1)
	}

	args := flag.Args()
	text := strings.Join(args[1:], " ")
	topic := args[0]
	m := tat.MessageJSON{
		Text:  text,
		Topic: topic,
	}

	fmt.Printf("Send on topic %s this message: %s\n", topic, text)

	msgCreated, err := client.MessageAdd(m)
	if err != nil {
		fmt.Printf("Error:%s\n", err)
		os.Exit(1)
	}
	fmt.Printf("ID Message Created: %s\n", msgCreated.Message.ID)
}

```

## Notice
This is just a simple example. Please do not use tat password in argument of your binary.
Please check next chapter <a href="../golang-full-example">"Full Example"</a> for a real example.
