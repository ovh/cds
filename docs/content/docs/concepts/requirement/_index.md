---
title: "Requirement"
weight: 8
card: 
  name: worker-model
  weight: 10
---

Type of requirements:

- Binary
- Model
- Hostname
- [Network access]({{< relref "/docs/concepts/requirement/requirement_network.md" >}})
- [Service]({{< relref "/docs/concepts/requirement/requirement_service.md" >}})
- Memory
- [OS & Architecture]({{< relref "/docs/concepts/requirement/requirement_os_arch.md" >}})

A [Job]({{< relref "/docs/concepts/job.md" >}}) will be executed by a **worker**.

CDS will choose and provision a worker for dependending on the **requirements** you define on your job.

You can set as many requirements as you want, following these rules:

- Only one model can be set as requirement
- Only one hostname can be set as requirement
- Only one OS & Architecture requirement can be set as at a time
- Memory and Services requirements are available only on Docker models
