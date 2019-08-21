---
title: "show"
notitle: true
notoc: true
---
# cdsctl environment variable show

`Show a CDS environment variable`

## Synopsis

`Show a CDS environment variable`

```
cdsctl environment variable show [ PROJECT-KEY ] ENV-NAME VARIABLE-NAME [flags]
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

* [cdsctl environment variable](/docs/components/cdsctl/environment/variable/)	 - `Manage CDS environment variables`

