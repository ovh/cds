---
title: "Requirement"
weight: 8
card:
  name: concept_worker-model
  weight: 3
---

Requirement types:

- Binary
- Model
- Hostname
- [Service]({{< relref "/docs/concepts/requirement/requirement_service.md" >}})
- [Memory]({{< relref "/docs/concepts/requirement/requirement_memory.md" >}})
- [OS & Architecture]({{< relref "/docs/concepts/requirement/requirement_os_arch.md" >}})
- [Region]({{< relref "/docs/concepts/requirement/requirement_region.md" >}})

A [Job]({{< relref "/docs/concepts/job.md" >}}) will be executed by a **worker**.

CDS will choose a worker dependending on the **requirements** you define for your job.

You can set as many requirements as you want, following these rules:

- Only one model can be set as requirement
- Only one hostname can be set as requirement
- Only one OS & Architecture requirement can be set at a time
- Memory and Services requirements are available only on Docker models
- Only one region can be set as requirement
