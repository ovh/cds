+++
title = "Command Line"
weight = 6

+++

## Download

You'll find last release of `cdsctl` on [Github Releases](https://github.com/ovh/cds/releases/latest).


## Authentication

Per default, the command line `cdsctl` uses your keychain on your os:

* OSX: Keychain Access
* Linux System: Secret-tool (libsecret) 
* Windows: Windows Credentials service

You can bypass keychain tools by using environment variables:

```bash
CDS_API_URL="https://instance.cds.api"  CDS_USER="username" CDS_TOKEN="yourtoken" cdsctl [command]
```

Want to debug something? You can use `CDS_VERBOSE` environment variable.

```bash
CDS_VERBOSE=true cdsctl [command]
```

If you're using a self-signed certificate on CDS API, you probably want to use `CDS_INSECURE` variable.

```bash
CDS_INSECURE=true cdsctl [command]
```