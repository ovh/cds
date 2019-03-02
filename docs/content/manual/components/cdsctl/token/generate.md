+++
title = "generate"
+++
## cdsctl token generate

`Generate a new token`

### Synopsis


Generate a new token when you use the cli or the api in scripts or for your worker, hatchery, uservices.

The expiration must be [daily|persistent|session].

Daily expirate after one day.

Persistent doesn't expirate until you revoke them.

Pay attention you must be an administrator of the group to launch this command.
	

```
cdsctl token generate GROUPNAME EXPIRATION [DESCRIPTION] [flags]
```

### Options

```
      --format string   Output format: plain|json|yaml (default "plain")
  -h, --help            help for generate
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

