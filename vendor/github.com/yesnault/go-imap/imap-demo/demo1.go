// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mxk/go-imap/imap"
)

const (
	Addr = "imap.example.org"
	User = "user@example.org"
	Pass = "mypassword"
	MBox = "GoIMAPxyz123"
)

const Msg = `
Subject: GoIMAP
From: GoIMAP <goimap@example.org>

hello, world

`

func main() {
	imap.DefaultLogger = log.New(os.Stdout, "", 0)
	imap.DefaultLogMask = imap.LogConn | imap.LogRaw

	c := Dial(Addr)
	defer func() { ReportOK(c.Logout(30 * time.Second)) }()

	if c.Caps["STARTTLS"] {
		ReportOK(c.StartTLS(nil))
	}

	if c.Caps["ID"] {
		ReportOK(c.ID("name", "goimap"))
	}

	ReportOK(c.Noop())
	ReportOK(Login(c, User, Pass))

	if c.Caps["QUOTA"] {
		ReportOK(c.GetQuotaRoot("INBOX"))
	}

	cmd := ReportOK(c.List("", ""))
	delim := cmd.Data[0].MailboxInfo().Delim

	mbox := MBox + delim + "Demo1"
	if cmd, err := imap.Wait(c.Create(mbox)); err != nil {
		if rsp, ok := err.(imap.ResponseError); ok && rsp.Status == imap.NO {
			ReportOK(c.Delete(mbox))
		}
		ReportOK(c.Create(mbox))
	} else {
		ReportOK(cmd, err)
	}
	ReportOK(c.List("", MBox))
	ReportOK(c.List("", mbox))
	ReportOK(c.Rename(mbox, mbox+"2"))
	ReportOK(c.Rename(mbox+"2", mbox))
	ReportOK(c.Subscribe(mbox))
	ReportOK(c.Unsubscribe(mbox))
	ReportOK(c.Status(mbox))
	ReportOK(c.Delete(mbox))

	ReportOK(c.Create(mbox))
	ReportOK(c.Select(mbox, true))
	ReportOK(c.Close(false))

	msg := []byte(strings.Replace(Msg[1:], "\n", "\r\n", -1))
	ReportOK(c.Append(mbox, nil, nil, imap.NewLiteral(msg)))

	ReportOK(c.Select(mbox, false))
	ReportOK(c.Check())

	fmt.Println(c.Mailbox)

	cmd = ReportOK(c.UIDSearch("SUBJECT", c.Quote("GoIMAP")))
	set, _ := imap.NewSeqSet("")
	set.AddNum(cmd.Data[0].SearchResults()...)

	ReportOK(c.Fetch(set, "FLAGS", "INTERNALDATE", "RFC822.SIZE", "BODY[]"))
	ReportOK(c.UIDStore(set, "+FLAGS.SILENT", imap.NewFlagSet(`\Deleted`)))
	ReportOK(c.Expunge(nil))
	ReportOK(c.UIDSearch("SUBJECT", c.Quote("GoIMAP")))

	fmt.Println(c.Mailbox)

	ReportOK(c.Close(true))
	ReportOK(c.Delete(mbox))
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

func Login(c *imap.Client, user, pass string) (cmd *imap.Command, err error) {
	defer c.SetLogMask(Sensitive(c, "LOGIN"))
	return c.Login(user, pass)
}

func Sensitive(c *imap.Client, action string) imap.LogMask {
	mask := c.SetLogMask(imap.LogConn)
	hide := imap.LogCmd | imap.LogRaw
	if mask&hide != 0 {
		c.Logln(imap.LogConn, "Raw logging disabled during", action)
	}
	c.SetLogMask(mask &^ hide)
	return mask
}

func ReportOK(cmd *imap.Command, err error) *imap.Command {
	var rsp *imap.Response
	if cmd == nil {
		fmt.Printf("--- ??? ---\n%v\n\n", err)
		panic(err)
	} else if err == nil {
		rsp, err = cmd.Result(imap.OK)
	}
	if err != nil {
		fmt.Printf("--- %s ---\n%v\n\n", cmd.Name(true), err)
		panic(err)
	}
	c := cmd.Client()
	fmt.Printf("--- %s ---\n"+
		"%d command response(s), %d unilateral response(s)\n"+
		"%s %s\n\n",
		cmd.Name(true), len(cmd.Data), len(c.Data), rsp.Status, rsp.Info)
	c.Data = nil
	return cmd
}
