// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/mxk/go-imap/imap"
)

const Addr = "imap.gmail.com:993"

// CustomClient demonstrates how to add a new command to the IMAP client.
type CustomClient struct{ *imap.Client }

func NewCustomClient(c *imap.Client) CustomClient {
	c.CommandConfig["XYZZY"] = &imap.CommandConfig{States: imap.Login}
	return CustomClient{c}
}

func (c CustomClient) XYZZY() (cmd *imap.Command, err error) {
	if !c.Caps["XYZZY"] {
		return nil, imap.NotAvailableError("XYZZY")
	}
	return imap.Wait(c.Send("XYZZY"))
}

func main() {
	imap.DefaultLogger = log.New(os.Stdout, "", 0)
	imap.DefaultLogMask = imap.LogConn | imap.LogRaw

	c := NewCustomClient(Dial(Addr))
	defer c.Logout(30 * time.Second)

	if _, err := c.XYZZY(); err != nil {
		panic(err)
	}
}

func Dial(addr string) (c *imap.Client) {
	var err error
	if strings.HasSuffix(addr, ":993") {
		c, err = imap.DialTLS(addr, nil)
	} else {
		c, err = imap.Dial(addr)
	}
	if err != nil {
		panic(err)
	}
	return c
}
