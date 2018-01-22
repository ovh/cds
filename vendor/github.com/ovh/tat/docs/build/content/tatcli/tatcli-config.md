---
title: "tatcli config -h"
weight: 2
toc: true
prev: "/tatcli/general"
next: "/tatcli/tatcli-group"

---

## Command Description
### tatcli config -h

```
Config commands: tatcli config <command>

Usage:
  tatcli config [command]

Aliases:
  config, c


Available Commands:
  template    Write a template configuration file in $HOME/.tatcli/config.json: tatcli config template
  show        Show Configuration: tatcli config show

Flags:
  -h, --help=false: help for config

Global Flags: see tatcli -h

Use "tatcli config [command] --help" for more information about a command.

```

## Example
### Credentials

Config file is under $HOME/.tatcli/config.json
You can create it with this command:
```bash
tatcli config template
```

Template is:
```
{
  "username":"myUsername",
  "password":"myPassword",
  "url":"http://urltat:port"
}
```
