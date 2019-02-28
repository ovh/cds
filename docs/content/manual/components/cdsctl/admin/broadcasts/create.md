+++
title = "create"
+++
## cdsctl admin broadcasts create

`Create a CDS broadcast`

### Synopsis

`Create a CDS broadcast`

```
cdsctl admin broadcasts create TITLE [flags]
```

### Examples

```
level info:

	cdsctl admin broadcasts create "the title" < content.md

level warning:

	cdsctl admin broadcasts create --level warning "the title" "the content"
	
```

### Options

```
  -h, --help           help for create
  -l, --level string   Level of broadcast: info or warning (default "info")
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl admin broadcasts](/cli/cdsctl/admin/broadcasts/)	 - `Manage CDS broadcasts`

