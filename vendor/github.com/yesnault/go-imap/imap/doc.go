// Copyright 2013 The Go-IMAP Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package imap implements an IMAP4rev1 client, as defined in RFC 3501.

The implementation provides low-level access to all protocol features described
in the relevant RFCs (see list below), and assumes that the developer is
familiar with the basic rules governing connection states, command execution,
server responses, and IMAP data types. Reading this documentation alone is not
sufficient for writing a working IMAP client. As a starting point, you should
read RFC 2683 to understand some of the nuances of the protocol operation.

The rest of the documentation deals with the implementation of this package and
not the protocol in general.

Introduction

The package provides three main objects for interacting with an IMAP4 server:
Client, Command, and Response. The client sends commands to the server and
receives responses. The Response object is capable of representing all possible
server responses and provides helper methods for decoding common data formats,
such as LIST, FETCH, SEARCH, etc.

The client has two interfaces for issuing commands. The Send method is the raw
command interface that can be used for implementing new commands, which are not
already supported by the package. All standard commands, as well as those from a
number of popular extensions, have dedicated methods that perform capability and
field type checks, and properly encode the command arguments before passing them
to Send.

Response Delivery

To support execution of multiple concurrent commands, each server response goes
through a filtering process to identify its owner. Each command in progress has
an associated ResponseFilter function for this purpose. The Client calls all
active filters in the command-issue order until one of the filters "claims" the
response. Claimed responses are appended to the Command.Data queue of the
claimer. Responses rejected by all filters are referred to as "unilateral server
data" and are appended to the Client.Data queue. Commands documented as
expecting "no specific responses" use a nil ResponseFilter, which never claims
anything. Thus, responses for commands such as NOOP are always delivered to
Client.Data queue.

The Client/Command state can only be updated by a call to Client.Recv. Each call
receives and delivers at most one response, but these calls are often implicit,
such as when using the Wait helper function (see below). Be sure to inspect and
clear out the Client data queue after all receive operations to avoid missing
important server updates. The Client example below demonstrates correct response
handling.

Concurrency

The Client and its Command objects cannot be used concurrently from multiple
goroutines. It is safe to pass Response objects to other goroutines for
processing, but the Client assumes "single-threaded" model of operation, so all
method calls for the same connection must be serialized with sync.Mutex or some
other synchronization mechanism. Likewise, it is not safe to access Client.Data
and Command.Data in parallel with a call that can append new responses to these
fields.

Asynchronous Commands

Unless a command is marked as being "synchronous", which is usually those
commands that change the connection state, the associated method returns as soon
as the command is sent to the server, without waiting for completion. This
allows the client to issue multiple concurrent commands, and then process the
responses and command completions as they arrive.

A call to Command.Result on a command that is "in progress" will block until
that command is finished. There is also a convenience function that turns any
asynchronous command into a synchronous one:

	cmd, err := imap.Wait(c.Fetch(...))

If err is nil when the call returns, the command was completed with the OK
status and all data responses (if any) are queued in cmd.Data.

Logging Out

The Client launches a goroutine to support receive operations with timeouts. The
user must call Client.Logout to close the connection and stop the goroutine. The
only time it is unnecessary to call Client.Logout is when the server closes the
connection first and Client.Recv returns io.EOF error.

RFCs

The following RFCs are implemented by this package:

	http://tools.ietf.org/html/rfc2087 -- IMAP4 QUOTA extension
	http://tools.ietf.org/html/rfc2088 -- IMAP4 non-synchronizing literals
	http://tools.ietf.org/html/rfc2177 -- IMAP4 IDLE command
	http://tools.ietf.org/html/rfc2971 -- IMAP4 ID extension
	http://tools.ietf.org/html/rfc3501 -- INTERNET MESSAGE ACCESS PROTOCOL - VERSION 4rev1
	http://tools.ietf.org/html/rfc3516 -- IMAP4 Binary Content Extension
	http://tools.ietf.org/html/rfc3691 -- Internet Message Access Protocol (IMAP) UNSELECT command
	http://tools.ietf.org/html/rfc4315 -- Internet Message Access Protocol (IMAP) - UIDPLUS extension
	http://tools.ietf.org/html/rfc4616 -- The PLAIN Simple Authentication and Security Layer (SASL) Mechanism
	http://tools.ietf.org/html/rfc4959 -- IMAP Extension for Simple Authentication and Security Layer (SASL) Initial Client Response
	http://tools.ietf.org/html/rfc4978 -- The IMAP COMPRESS Extension
	http://tools.ietf.org/html/rfc5161 -- The IMAP ENABLE Extension
	http://tools.ietf.org/html/rfc5738 -- IMAP Support for UTF-8

The following RFCs are either informational, not fully implemented, or place no
implementation requirements on the package, but may be relevant to other parts
of a client application:

	http://tools.ietf.org/html/rfc2595 -- Using TLS with IMAP, POP3 and ACAP
	http://tools.ietf.org/html/rfc2683 -- IMAP4 Implementation Recommendations
	http://tools.ietf.org/html/rfc4466 -- Collected Extensions to IMAP4 ABNF
	http://tools.ietf.org/html/rfc4469 -- Internet Message Access Protocol (IMAP) CATENATE Extension
	http://tools.ietf.org/html/rfc4549 -- Synchronization Operations for Disconnected IMAP4 Clients
	http://tools.ietf.org/html/rfc5530 -- IMAP Response Codes
*/
package imap
