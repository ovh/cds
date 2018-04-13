+++
title = "Workflow"
weight = 2

+++

![Workflow](/images/concepts_workflow.png)

A workflow allows you to:

* chain pipelines with triggers
* add hooks to trigger pipeline

When you run a workflow, CDS will take a snapshot of it for your execution. 
That means that if you modify one pipeline (or application or environment) after, this will not affect your workflow execution.
Nevertheless, CDS will allow you to resynchronize all contexts linked to a workflow execution and re-run some part of it with up to date datas.