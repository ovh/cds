---
title: "disable"
notitle: true
notoc: true
---
# cdsctl worker disable

`Disable CDS workers`

## Synopsis

Disable one on more CDS worker by their names.

For example if your want to disable all CDS workers you can run:

$ cdsctl worker disable $(cdsctl worker list)

```
cdsctl worker disable NAME ...
```

## Options inherited from parent commands

```
  -f, --file string   set configuration file
      --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
      --verbose       verbose output
```

## SEE ALSO

* [cdsctl worker](/docs/components/cdsctl/worker/)	 - `Manage CDS worker`

