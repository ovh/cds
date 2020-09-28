---
title: "Mutex"
weight: 6
---

By default, the same pipeline can be run multiple times at once.

In a CDS Workflow, you can limit running a pipeline to one at a time.

Click on the pipeline  → Edit the pipeline context → enable  "Limit one run at run time"

![Pipeline Mutex](/images/workflows.design.mutex.png)

Examplary use case: run an integration test once on a particular environment.

To configure a Mutex with the configuration as code, use the boolean property `one_at_a_time`
in the workflow definition file, in a pipeline context section:
[Mutex configuration as code example]({{<relref "/docs/concepts/files/workflow-syntax.md#mutex">}}).
