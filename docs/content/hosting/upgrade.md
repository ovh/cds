---
title: "Upgrade your CDS Instance"
weight: 9
card: 
  name: operate
---

## Upgrade Binary

Update your CDS Engine binary from latest Release from GitHub:

```bash
./engine update --from-github
```

## Database Migration

```bash
# get the file sql.tar.gz from latest release from https://github.com/ovh/cds/releases
# unzip sql.tar.gz inside a sql/ directory, then run this command:
./engine database upgrade --db-password=cds --db-sslmode=disable --db-name=cds --db-schema=public --migrate-dir=sql/api --db-connect-timeout=20
./engine database upgrade --db-password=cds --db-sslmode=disable --db-name=cds --db-schema=cdn --migrate-dir=sql/cdn --db-connect-timeout=20
```

## Restart your CDS API

```bash
./engine start api ... 
```
