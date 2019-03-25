---
title: "new"
notitle: true
notoc: true
---
# cdsctl xtoken new

`Create a new access token`

## Synopsis

`Create a new access token`

```
cdsctl xtoken new [flags]
```

## Options

```
  -d, --description string   what is the purpose of this token
  -e, --expiration string    expiration delay of the token (1d, 24h, 1440m, 86400s) (default "1d")
  -g, --group strings        define the scope of the token through groups
```

## Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

## SEE ALSO

* [cdsctl xtoken](/docs/components/cdsctl/xtoken/)	 - `Manage CDS access tokens [EXPERIMENTAL]`

