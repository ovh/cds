---
title: "API - System"
weight: 6
toc: true
prev: "/engine/api-presences"
next: "/engine/simple-installation"

---


## System
### Version

```bash
curl -XGET https://<tatHostname>:<tatPort>/version
```


## Stats

For Tat admin only.

### Count

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/stats/count
```

### Instance

Info about current instance of engine

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/stats/instance
```

### Distribution

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/stats/distribution
```

### DB Stats

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/stats/db/stats
```

### DB ServerStatus

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/stats/db/serverStatus
```

### DB Replica Set Status

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/stats/db/replSetGetStatus
```

### DB Replica Set Config

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/stats/db/replSetGetConfig
```



### DB Stats of each collections

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/stats/db/collections
```

### DB Stats slowest Queries

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/stats/db/slowestQueries
```

## System
### Capabilities

Return `websocket-enabled` and `username-from-email` parameters. See Tat Flags below.
```bash
curl -XGET https://<tatHostname>:<tatPort>/capabilities
```

### Flush Cache
```bash
curl -XGET https://<tatHostname>:<tatPort>/system/cache/clean
```

### Cache Info
```bash
curl -XGET https://<tatHostname>:<tatPort>/system/cache/info
```
