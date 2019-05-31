---
title: "JUnit"
card:
  name: builtin
---

**JUnit** is a builtin action, you can't modify it.

This action parses a given Junit formatted XML file to extract its test results.

## Parameters

* **path**: Path to junit xml file.


## Requirements

No Requirement

## YAML example

Example of a pipeline using JUnit action:
```yml
version: v1.0
name: Pipeline1
stages:
- Stage1
jobs:
- job: Job1
  stage: Stage1
  steps:
  - jUnitReport: '{{.cds.workspace}}/report.xml'

```

## Example

* Job Configuration.

![img](/images/workflows.pipelines.actions.builtin.junit-job.png)

* Launch pipeline, check XUnit Result

![img](/images/workflows.pipelines.actions.builtin.junit-view.png)

* And view details:

![img](/images/workflows.pipelines.actions.builtin.junit-view-details.png)
