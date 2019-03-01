+++
title = "Pipeline Context"
weight = 2

+++

After adding the pipeline, you can "Edit the pipeline Context" (sidebar).

![Select Pipeline](/images/workflows.design.ctx.select.png)

Then, you can: 

* add or remove application. Jobs can use `cds.app.*` [configuration]({{< relref "manual/concepts/variables.md">}})
* and or remove an environment. Jobs can use `cds.env.*` [configuration]({{< relref "manual/concepts/variables.md">}})
* enable / disable Pipeline Mutex

![Pipeline Edit Context](/images/workflows.design.ctx.edit.png)
