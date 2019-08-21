---
title: "find"
notitle: true
notoc: true
---
# cdsctl token find

`Find an existing token`

## Synopsis


Find an existing token with his value to have his description, creation date and the name of the creator.
	

```
cdsctl token find TOKEN [flags]
```

## Examples

```
cdsctl token find "myTokenValue"
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

