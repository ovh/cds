---
title: "DeployApplication"
card:
  name: builtin
---

**DeployApplication** is a builtin action, you can't modify it.

Deploy an application, useful only if you have a Deployment Plaftorm associated to your current application.

## Parameters

No Parameter

## Requirements

No Requirement

## YAML example

Example of a pipeline using DeployApplication action:
```yml
version: v1.0
name: Pipeline1
stages:
- Stage1
jobs:
- job: Job1
  stage: Stage1
  steps:
  - deploy: '{{.cds.application}}'

```

## Example

* Add a deployment platform on your application.

![img](/images/workflows.pipelines.actions.builtin.deploy-application-1.png)

* Create a workflow, add a pipeline and an application linked to a platform.

![img](/images/workflows.pipelines.actions.builtin.deploy-application-2.png)

* Or edit the pipeline context from your workflow view.

![img](/images/workflows.pipelines.actions.builtin.deploy-application-3.png)

* In the job, use action DeployApplication

![img](/images/workflows.pipelines.actions.builtin.deploy-application-4.png)
