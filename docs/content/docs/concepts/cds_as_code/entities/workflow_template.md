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

### Workflow name

`[[ .name ]]` gives the name of the workflow that uses the template:

```yaml
spec: |-
  jobs:
    deploy:
      runs-on: my-model
      steps:
      - run: echo "running [[.name]]"
```

`${{ cds.workflow }}` carries the same value, but the two are not interchangeable. `[[ .name ]]` is
resolved when the run is crafted, so it can be used where the value is needed before the workflow is
built: job keys, stage names, `on:`. `${{ cds.workflow }}` is only interpolated when the job runs.

Writing `name:` in a `spec:` has no effect: the workflow keeps the name of the file that calls the
template, so `[[ .name ]]` is read-only.

### Variable sets

To know whether a [variable set](/docs/concepts/cds_as_code/project/variableset/) exists on the
project, pass its name to `.vars.Exists`:

```yaml
spec: |-
  jobs:
    build:
      runs-on: my-model
    [[- if .vars.Exists "deploy-creds" ]]
    deploy:
      runs-on: my-model
      vars: [deploy-creds]
      steps:
      - run: echo "${{ vars.deploy-creds.TOKEN }}"
    [[- end ]]
```

When the variable set is missing, the guarded block is not generated at all. The job is absent from
the workflow, it is not a skipped job.

Combine several variable sets with `and`:

```yaml
    [[- if and (.vars.Exists "deploy-creds") (.vars.Exists "deploy-regions") ]]
```

`.vars` answers existence and nothing else. It exposes neither the items of a variable set nor their
values, which are read at runtime with `${{ vars.<varset_name>.<item_name> }}`.

Two limitations:

- `on:` must not be driven by `.vars`. Hooks are computed when the repository is analyzed, and an
  analysis is not replayed when a variable set is created or deleted.
- the resolution happens when the run is crafted. Creating or deleting a variable set leaves a run
  that already started untouched.

## Previewing a template

`cdsctl experimental template generate-from-file` renders a template file without pushing it. It has
neither a project nor a workflow to read from, so `--vars` simulates the variable sets and `--name`
supplies the workflow name that `[[ .name ]]` returns:

```bash
cdsctl experimental template generate-from-file my-template.yml \
  -p env=prod --vars deploy-creds --name my-workflow
```

Without `--name`, `[[ .name ]]` renders empty.
