---
title: "delete"
notitle: true
notoc: true
---
# cdsctl worker model delete

`Delete a CDS worker model`

## Synopsis

`Delete a CDS worker model`

```
cdsctl worker model delete WORKER-MODEL-PATH [flags]
```

## Examples

```
cdsctl worker model delete shared.infra/myModel
```

## Options

```
      --force   Force delete without confirmation and exit 0 if resource does not exist
```

## Options inherited from parent commands

```
  -f, --file string   set configuration file
      --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
      --verbose       verbose output
```

## SEE ALSO

* [cdsctl worker model](/docs/components/cdsctl/worker/model/)	 - `Manage Worker Model`

