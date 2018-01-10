+++
title = "Mutex"
weight = 7

[menu.main]
parent = "design"
identifier = "design.mutex"

+++

By default, the same pipeline can run at the same time on multiple runs.

In a CDS Workflow, you can limit running a pipeline one at a time. 

Click on the pipeline  → Edit the pipeline context → enable  "Limit one run at run time"

![Pipeline Mutex](/images/workflows.design.mutex.png)

Example of use case: run an integration test once on a particular environment.

