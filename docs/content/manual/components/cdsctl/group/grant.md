+++
title = "grant"
+++
## cdsctl group grant

`Grant a CDS group in a project or workflow`

### Synopsis

`Grant a CDS group in a project or workflow`

```
cdsctl group grant [ PROJECT-KEY ] GROUP-NAME PERMISSION [flags]
```

### Options

```
  -h, --help              help for grant
  -p, --only-project      Indicate if the group must be added only on project or also on all workflows in project
  -n, --workflow string   Workflow name
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl group](/manual/components/cdsctl/group/)	 - `Manage CDS group`

