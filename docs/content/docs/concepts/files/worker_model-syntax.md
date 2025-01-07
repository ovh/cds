---
title: "Worker Model configuration file"
weight: 2
card: 
  name: concept_worker-model
---

Example:

```yml
name: go-official-1.23
group: shared.infra
image: golang:1.23
description: official from https://hub.docker.com/_/golang/
type: docker
pattern_name: basic_unix
```

Import a worker model:

```bash
cdsctl worker model import ./go-official-1.23.yml
```

or with a remote file:

```bash
cdsctl worker model import https://raw.githubusercontent.com/ovh/cds/{{< param "version" "master" >}}/contrib/worker-models/go-official-1.23.yml
```

{{< note >}}
If you want to specify an image using a private registry or a private image, you need to fill credentials in field `username` and `password` to access your image. And if your image is not on docker hub but from a private registry, you need to fill the `registry` info (the registry api url, for example for docker hub it's https://index.docker.io/v1/ but we fill it by default).
{{< /note >}}
