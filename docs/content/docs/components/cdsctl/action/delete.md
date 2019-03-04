---
title: "delete"
notitle: true
notoc: true
---
# cdsctl action delete

`Delete a CDS action`

## Synopsis

Useful to delete a CDS action

	cdsctl action delete myAction

	# this will not fail if action does not exist
	cdsctl action delete myActionNotExist --force


```
cdsctl action delete ACTION-NAME [flags]
```

## Options

```
      --force   if true, do not fail if action does not exist
```

## Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

## SEE ALSO

* [cdsctl action](/docs/components/cdsctl/action/)	 - `Manage CDS action`

