---
title: "Workflow Template"
weight: 2
---

# Description

Workflow template is a CDS entity that allows you to template a workflow. It allows you to run a workflow with parameters.

# As Code directory

A workflow is described directly on your repository inside the directory `.cds/workflow-templates/`.

# Permission

The permission `manage-workflow-template` on your project is mandatory to manage a workflow.

# Fields

```yaml
name: workflow-template
parameters:
  - key: var1
  - key: var2
spec: |-
  commit-status: ...
  on: [push]
  integrations: [my-artifactory]
  jobs:
    job1:
      runs-on:
        model: "{{.worker_model}}"
        memory: "512"
      steps:
      - run: |-
        #!/bin/bash
        env
        echo "[[.params.var1]]"
      [[- if .params.var2 ]]
      - run: |-
        #!/bin/bash
        echo "[[.params.var2]]"
      [[- end ]]
  env:
    VAR_1: value
    VAR_2: value2
  stages: ...
  gates: ...
```

- <span style="color:red">\*</span>`name`: The name of your workflow template
- [`parameters`](#parameters): Input parameters for the template, accssible with `.params`
- <span style="color:red">\*</span>[`spec`](#spec): Template of the workflow

<span style="color:red">\*</span> mandatory fields

## Parameters

Input parameters for the workflow template.

```yaml
key: varname
required: true
```

- <span style="color:red">\*</span>`key`: Name of the parameter
- `required`: Indicate if the parameter is mandatory

## Spec

Spec is a text-based field that expect a raw spec of a workflow.
Templating is done using golang template engine and delimiters used are `[[` and `]]`.

To access input parameters set from the workflow, use `.params`.

```yaml
spec: |-
  commit-status: ...
  on: [push]
  integrations: [my-artifactory]
  jobs:
    job1:
      runs-on:
        model: "{{.worker_model}}"
        memory: "512"
      steps:
      - run: |-
        #!/bin/bash
        env
        echo "[[.params.var1]]"
      [[- if .params.var2 ]]
      - run: |-
        #!/bin/bash
        echo "[[.params.var2]]"
      [[- end ]]
  env:
    VAR_1: value
    VAR_2: value2
  stages: ...
  gates: ...
```
