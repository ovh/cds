+++
title = "Monitoring"
weight = 8

+++

### Status Handler on API

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

### Monitoring with Command Line

```bash
./cdsctl monitoring
```

This will returns Queue status, Workers & Hatheries Status and CDS Engine Status on bottom right.

![cdsctl monitoring](/images/hosting.monitoring.png)