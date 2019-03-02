+++
title = "find"
+++
## cdsctl token find

`Find an existing token`

### Synopsis


Find an existing token with his value to have his description, creation date and the name of the creator.
	

```
cdsctl token find TOKEN [flags]
```

### Examples

```
cdsctl token find "myTokenValue"
```

### Options

```
      --format string   Output format: plain|json|yaml (default "plain")
  -h, --help            help for find
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl token](/manual/components/cdsctl/token/)	 - `Manage CDS group token`

