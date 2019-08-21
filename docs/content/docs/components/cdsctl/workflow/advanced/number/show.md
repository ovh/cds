---
title: "show"
notitle: true
notoc: true
---
# cdsctl workflow advanced number show

`Show a Workflow Run Number`

## Synopsis

`Show a Workflow Run Number`

```
cdsctl workflow advanced number show [ PROJECT-KEY WORKFLOW-NAME ] [flags]
```

## Examples

```
cdsctl workflow advanced number show MYPROJECT my-workflow
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

* [cdsctl workflow advanced number](/docs/components/cdsctl/workflow/advanced/number/)	 - `Manage Workflow Run Number`

