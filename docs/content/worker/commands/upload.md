+++
title = "Worker Upload"
weight = 5

+++

Inside a job, there are two ways to upload an artifact:

* with a step using action Upload Artifacts
* with a step [script]({{< relref "workflows/pipelines/actions/builtin/script.md" >}}), using the worker command: `worker upload --tag=<tag> <path>`


```bash
# worker upload --tag=<tag> <path>
worker export --tag={{.cds.version}} files*.yml
```
