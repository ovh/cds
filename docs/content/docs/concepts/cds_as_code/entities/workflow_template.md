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
- [`parameters`](#parameters): Input parameters for the template, accessible with `.params`
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

### Variable sets

To know whether a [variable set](/docs/concepts/cds_as_code/project/variableset/) exists on the
project, use `.vars.Exists`. It takes the variable set name, and optionally item names that must all
exist in it:

```yaml
spec: |-
  jobs:
    build:
      runs-on: my-model
    [[- if .vars.Exists "deploy-creds" ]]
    deploy:
      runs-on: my-model
      vars: [deploy-creds]
    [[- end ]]
    [[- if .vars.Exists "deploy-creds" "TOKEN" "REGION" ]]
    deploy-regional:
      runs-on: my-model
      vars: [deploy-creds]
      steps:
      - run: echo "${{ vars.deploy-creds.TOKEN }}"
    [[- end ]]
```

When the variable set is missing, the guarded block is not generated at all. The job is absent from
the workflow, it is not a skipped job.

`.vars` only tells whether a variable set and its items exist, it never exposes their values. Values
are read at runtime with `${{ vars.<varset_name>.<item_name> }}`.

Two limitations:

- `on:` must not be driven by `.vars`. Hooks are computed when the repository is analyzed, and an
  analysis is not replayed when a variable set is created or deleted.
- the resolution happens when the run is crafted. Creating or deleting a variable set leaves a run
  that already started untouched.

`cdsctl experimental template generate-from-file` has no project context. Simulate variable sets
there with `--vars`:

```bash
cdsctl experimental template generate-from-file my-template.yml -p env=prod \
  --vars deploy-creds.TOKEN --vars deploy-creds.REGION
```
