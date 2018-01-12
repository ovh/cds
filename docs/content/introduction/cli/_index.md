+++
title = "Command Line"
weight = 4

+++


## Download

You'll find last release of `cdsctl` on [Github Releases](https://github.com/ovh/cds/releases/latest).

## Commands

```
cdsctl -h
CDS Command line utility

Usage:
  cdsctl [flags]
  cdsctl [command]

Available Commands:
  action      Manage CDS action
  login       Login to CDS
  signup      Signup on CDS
  application Manage CDS application
  environment Manage CDS environment
  pipeline    Manage CDS pipeline
  group       Manage CDS group
  health      Check CDS health
  project     Manage CDS project
  worker      Manage CDS worker
  workflow    Manage CDS workflow
  update      Update cdsctl from CDS API or from CDS Release
  user        Manage CDS user
  monitoring  CDS monitoring
  health      Check CDS health
  version     show cdsctl version

Flags:
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output

Use "cdsctl [command] --help" for more information about a command.

```