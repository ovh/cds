---
title: "tatcli stats -h"
weight: 6
toc: true
prev: "/tatcli/tatcli-presence"
next: "/tatcli/tatcli-system"

---

## Command Description
### tatcli stats -h

```
Stats commands (admin only): tatcli stats [<command>]

Usage:
  tatcli stats [command]

Aliases:
  stats, stat


Available Commands:
  count              Count all messages, groups, presences, users, groups, topics: tatcli stats count
  distribution       Distribution of messages per topics: tatcli stats distribution
  dbstats            DB Stats: tatcli stats dbstats
  dbServerStatus     DB Stats: tatcli stats dbServerStatus
  dbReplSetGetConfig DB Stats: tatcli stats dbReplSetGetConfig
  dbReplSetGetStatus DB Stats: tatcli stats dbReplSetGetStatus
  dbCollections      DB Stats on each collection: tatcli stats dbCollections
  dbSlowestQueries   DB Stats slowest Queries: tatcli stats dbSlowestQueries
  instance           Info about current instance of engine

Flags:
  -h, --help=false: help for stats

Global Flags: see tatcli -h

```
