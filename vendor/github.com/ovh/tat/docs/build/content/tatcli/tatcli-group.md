---
title: "tatcli group -h"
weight: 3
toc: true
prev: "/tatcli/tatcli-config"
next: "/tatcli/tatcli-message"

---

## Command Description

### tatcli group -h

```
Group commands: tatcli group <command>

Usage:
  tatcli group [command]

Aliases:
  group, g


Available Commands:
  list            List all groups: tatcli group list <skip> <limit>
  create          create a new group: tatcli group create <groupname> <description>
  update          update a group: tatcli group update <groupname> <newGroupname> <newDescription>
  delete          delete a group: tatcli group delete <groupname>
  addUser         Add Users to a group: tacli group addUser <groupname> <username1> [<username2> ... ]
  deleteUser      Delete Users from a group: tacli group deleteUser <groupname> <username1> [<username2> ... ]
  addAdminUser    Add Admin Users to a group: tacli group addAdminUser <groupname> <username1> [<username2> ... ]
  deleteAdminUser Delete Admin Users from a group: tacli group deleteAdminUser <groupname> <username1> [<username2> ... ]

Flags:
  -h, --help=false: help for group

Global Flags: see tatcli -h

Use "tatcli group [command] --help" for more information about a command.

```

## Examples

### Create a group (Admin only)
```bash
tatcli group add groupname description of group
```

### Update a group (Admin only)
```bash
tatcli group update groupname newGroupname new description of group
```

### Delete a group (Admin only)
```bash
tatcli group delete groupname
```

### Add user to a group
```bash
tatcli group addUser groupname username
```

### Delete a user from a group
```bash
tatcli group deleteUser groupname username
```
