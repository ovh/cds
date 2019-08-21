---
title: "generate"
notitle: true
notoc: true
---
# cdsctl token generate

`Generate a new token`

## Synopsis


Generate a new token when you use the cli or the api in scripts or for your worker, hatchery, µservices.

The expiration must be [daily|persistent|session].

Daily expirate after one day.

Persistent doesn't expirate until you revoke them.

Pay attention you must be an administrator of the group to launch this command.
	

```
cdsctl token generate GROUPNAME EXPIRATION [DESCRIPTION] [flags]
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

* [cdsctl token](/docs/components/cdsctl/token/)	 - `Manage CDS group token`

