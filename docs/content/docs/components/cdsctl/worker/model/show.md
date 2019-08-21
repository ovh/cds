---
title: "show"
notitle: true
notoc: true
---
# cdsctl worker model show

`Show a Worker Model`

## Synopsis

`Show a Worker Model`

```
cdsctl worker model show WORKER-MODEL-PATH [flags]
```

## Examples

```
cdsctl worker model show myGroup/myModel
```

## Options

```
      --fields string   Only display specified object fields. 'empty' will display all fields, 'all' will display all object fields, 'field1,field2' to select multiple fields
      --format string   Output format: plain|json|yaml (default "plain")
  -q, --quiet           Only display object's key
```

## Options inherited from parent commands

```
  -f, --file string   set configuration file
      --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
      --verbose       verbose output
```

## SEE ALSO

* [cdsctl worker model](/docs/components/cdsctl/worker/model/)	 - `Manage Worker Model`

