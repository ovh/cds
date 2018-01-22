// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imap_test

import (
	"bytes"
	"fmt"
	"net/mail"
	"time"

	"github.com/mxk/go-imap/imap"
)

func ExampleClient() {
	//
	// Note: most of error handling code is omitted for brevity
	//
	var (
		c   *imap.Client
		cmd *imap.Command
		rsp *imap.Response
	)

	// Connect to the server
	c, _ = imap.Dial("imap.example.com")

	// Remember to log out and close the connection when finished
	defer c.Logout(30 * time.Second)

	// Print server greeting (first response in the unilateral server data queue)
	fmt.Println("Server says hello:", c.Data[0].Info)
	c.Data = nil

	// Enable encryption, if supported by the server
	if c.Caps["STARTTLS"] {
		c.StartTLS(nil)
	}

	// Authenticate
	if c.State() == imap.Login {
		c.Login("user@example.com", "mysupersecretpassword")
	}

	// List all top-level mailboxes, wait for the command to finish
	cmd, _ = imap.Wait(c.List("", "%"))

	// Print mailbox information
	fmt.Println("\nTop-level mailboxes:")
	for _, rsp = range cmd.Data {
		fmt.Println("|--", rsp.MailboxInfo())
	}

	// Check for new unilateral server data responses
	for _, rsp = range c.Data {
		fmt.Println("Server data:", rsp)
	}
	c.Data = nil

	// Open a mailbox (synchronous command - no need for imap.Wait)
	c.Select("INBOX", true)
	fmt.Print("\nMailbox status:\n", c.Mailbox)

	// Fetch the headers of the 10 most recent messages
	set, _ := imap.NewSeqSet("")
	if c.Mailbox.Messages >= 10 {
		set.AddRange(c.Mailbox.Messages-9, c.Mailbox.Messages)
	} else {
		set.Add("1:*")
	}
	cmd, _ = c.Fetch(set, "RFC822.HEADER")

	// Process responses while the command is running
	fmt.Println("\nMost recent messages:")
	for cmd.InProgress() {
		// Wait for the next response (no timeout)
		c.Recv(-1)

		// Process command data
		for _, rsp = range cmd.Data {
			header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
			if msg, _ := mail.ReadMessage(bytes.NewReader(header)); msg != nil {
				fmt.Println("|--", msg.Header.Get("Subject"))
			}
		}
		cmd.Data = nil

		// Process unilateral server data
		for _, rsp = range c.Data {
			fmt.Println("Server data:", rsp)
		}
		c.Data = nil
	}

	// Check command completion status
	if rsp, err := cmd.Result(imap.OK); err != nil {
		if err == imap.ErrAborted {
			fmt.Println("Fetch command aborted")
		} else {
			fmt.Println("Fetch error:", rsp.Info)
		}
	}
}
