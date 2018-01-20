---
title: "Simple Installation"
weight: 41
toc: true
prev: "/engine/api-system"
next: "/engine/production-installation"

---

![Installation Simple Way](/imgs/tat-installation-simple.png?width=50%)

# Prerequisites
* MongoDB >= 3.2
* Download latest release of Tat
 * OSX https://github.com/ovh/tat/releases/download/RELEASE_VERSION/tat-darwin-amd64
 * Linux https://github.com/ovh/tat/releases/download/RELEASE_VERSION/tat-linux-amd64
 * replace RELEASE_VERSION with latest release from https://github.com/ovh/tat/releases

# Run in Development Mode
* Start Tat without SMTP Server. Mails sent will be displayed in console.

```bash
$ mv tat-<architecture> tat
$ chmod +x tat
$ ./tat --no-smtp
```

Output logs:
```bash
./api --no-smtp
[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:	export GIN_MODE=release
 - using code:	gin.SetMode(gin.ReleaseMode)

INFO[0000] Mongodb : create new instance

[...] # DEBUG Logs here

INFO[0000] No Kafka configured
INFO[0000] TAT is running on 8080
INFO[0000] TAT is NOT linked to a redis
# Tat is ready
```

Try it
```
$ curl localhost:8080/version
{"version":"2.0.0"}
```
