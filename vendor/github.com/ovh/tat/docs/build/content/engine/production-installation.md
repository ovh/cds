---
title: "Installation - Production Way"
weight: 42
toc: true
prev: "/engine/simple-installation"

---

Don't worry, this installation describes below is used @ OVH to handle at least:

* 800.000 GET / 2 hours
* 50.000 POST / 2 hours
* 30.000 DELETE / 2 hours
* 45.000 PUT / 2 hours

![Installation Production Way](/imgs/tat-installation-production.png?width=75%)

# Prerequisites
* MongoDB >= 3.2, with ReplicaSet
* Redis with Slave and Sentinel
* Download latest release of Tat
 * OSX https://github.com/ovh/tat/releases/download/RELEASE_VERSION/tat-darwin-amd64
 * Linux https://github.com/ovh/tat/releases/download/RELEASE_VERSION/tat-linux-amd64
 * replace RELEASE_VERSION with latest release from https://github.com/ovh/tat/releases

# Run in Production Mode

```bash
#!/bin/bash

# Logs Configuration
TAT_PRODUCTION=true

TAT_LISTEN_PORT=8080

# Default group for each new user
TAT_DEFAULT_GROUP=Common_Team

# HTTP Timeout Configuration
TAT_READ_TIMEOUT=55
TAT_WRITE_TIMEOUT=55

# SMTP Configuration
TAT_SMTP_HOST=your-smtp-host
TAT_SMTP_TLS=true
TAT_SMTP_PORT=25
TAT_SMTP_FROM=noreply.tat@your-tat-hostname

# Exposed var are used in emails for add / reset a user
TAT_EXPOSED_SCHEME=http
TAT_EXPOSED_PATH=/
TAT_EXPOSED_PORT=8080
TAT_EXPOSED_HOST=your-tat-hostname

# MongoDB Configuration
TAT_DB_USER=tat
TAT_DB_PASSWORD=your-mongodb-password
TAT_DB_ADDR=mongodb-master,mongodb-rs-1,mongodb-rs-2/tat
TAT_DB_SOCKET_TIMEOUT=60
TAT_DB_ENSURE_SAFE_DB_WRITE=3

# Redis Configuration for cache
TAT_REDIS_SENTINELS=redis-host:27001,redis-host:27002,redis-host:27003,redis-host:27004
TAT_REDIS_MASTER=tatredismaster
TAT_REDIS_PASSWORD=your-redis-password

# TAT 2 XMPP Configuration
TAT_TAT2XMPP_USERNAME=tat.system.jabber
TAT_TAT2XMPP_URL=http://tat2xmpp.your-domain
TAT_TAT2XMPP_KEY=a-key-used-by-tat2xmpp

./api
```
