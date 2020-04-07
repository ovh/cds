---
title: "Monitoring"
weight: 8
card: 
  name: operate
---

## Status Handler on API

https://your.cds.instance/mon/status returns the status of CDS Engine.

If status != OK, something is wrong on your CDS Instance.

Example:

```json
{
    "now": "2018-01-09T20:24:20.481193492Z",
    "lines": [ 
        { "status": "OK", "component": "Version", "value": "0.25.1-snapshot+1455.cds" },
    ...
        { "status": "OK", "component": "Database", "value": "20 conns" },
        { "status": "OK", "component": "LastUpdate Connected", "value": "14" },
        { "status": "OK", "component": "Worker Model Errors", "value": "0" }
    ...
}
```

## Monitoring with Command Line

```bash
# display the current job's queue
./cdsctl queue

# display the status of all service, except the status OK
./cdsctl -c prod health status --filter STATUS="[^O].*"
```
