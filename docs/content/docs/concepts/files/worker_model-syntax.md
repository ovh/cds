---
title: "Worker Model configuration file"
weight: 2
card: 
  name: concept_worker-model
---

Example:

```yml
name: go-official-1.11.4-stretch
group: shared.infra
communication: http
image: golang:1.11.4-stretch
description: official from https://hub.docker.com/_/golang/
type: docker
pattern_name: basic_unix
```

Import a worker model

```bash
cdsctl worker model import ./go-official-1.11.4-stretch.yml
```

or with a remote file:

```bash
cdsctl worker model import https://raw.githubusercontent.com/ovh/cds/master/contrib/worker-models/go-official-1.11.4-stretch.yml
```
