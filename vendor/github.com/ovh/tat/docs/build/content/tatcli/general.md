---
title: "General"
weight: 1
toc: true
prev: "/tatcli"
next: "/tatcli/tatcli-config"

---

<img align="right" src="https://raw.githubusercontent.com/ovh/tat/master/tat.png">

Tatcli, a Tat Command Line Interface.

See Tat Engine for more information: https://github.com/ovh/tat

## Download
Download latest binary on release page https://github.com/ovh/tat/releases 
then `chmod +x tatcli`

If you have already installed tatcli, you can update it with `tatcli update`.

## Usage - General Rules

A successful command will give you no feedback. If you want one, you can use `-v` argument.
After each command, the exit code can be found in the `$?` variable. No error if exit code equals 0.
