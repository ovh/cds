+++
title = "request"
+++
## cdsctl admin services request

`request GET on a CDS service`

### Synopsis

`request GET on a CDS service`

```
cdsctl admin services request [flags]
```

### Examples

```

## How to get the goroutine of the service named hatcheryLocal:
```bash
cdsctl admin services request --name hatcheryLocal --query /debug/pprof/goroutine\?debug\=2
```


```

### Options

```
  -h, --help           help for request
      --name string    service name
      --query string   http query, example: '/debug/pprof/goroutine?debug=2'
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl admin services](/manual/components/cdsctl/admin/services/)	 - `Manage CDS services`

