---
title: "status"
notitle: true
notoc: true
---
# cdsctl workflow status

`Check the status of the run`

## Synopsis

`Check the status of the run`

```
cdsctl workflow status [ PROJECT-KEY WORKFLOW-NAME ] [RUN-NUMBER] [flags]
```

## Options

```
      --fields string   Only display specified object fields. 'empty' will display all fields, 'all' will display all object fields, 'field1,field2' to select multiple fields
      --format string   Output format: plain|json|yaml (default "plain")
  -q, --quiet           Only display object's key
      --track           Wait the workflow to be over
```

## Options inherited from parent commands

```
  -f, --file string   set configuration file
      --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
      --verbose       verbose output
```

## SEE ALSO

* [cdsctl workflow](/docs/components/cdsctl/workflow/)	 - `Manage CDS workflow`

