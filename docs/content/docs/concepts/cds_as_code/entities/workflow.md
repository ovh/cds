---
title: "Workflow"
weight: 2
---

# Description

The workflow is the main entity in CDS. It allows you to chain jobs, using conditional branching.

# As Code directory

A workflow is described directly on your repository inside the directory `.cds/workflows/`.

# Permission

The permission `manage-workflow` on your project is mandatory to manage a workflow.

# Fields

```yaml
name: cds
repository:
  vcs: github
  name: ovh/cds
commit-status: ...
on: [push]
integrations: [my-artifactory]
stages: ...
jobs: ...
env:
  VAR_1: value
  VAR_2: value2
gates: ...
```

Workflow from a template

```yaml
name: cds
repository:
  vcs: github
  name: ovh/cds
commit-status: ...
on: [push]
from: .cds/workflow-templates/mtemplate.yml
parameters:
  - key: param1
    required: true
```

- <span style="color:red">\*</span>`name`: The name of your workflow
- <span style="color:red">\*</span>[`on`](#on): Allow you to define hooks to trigger your workflow
- <span style="color:red">\*</span>[`jobs`](#jobs): Jobs definitions
- [`integrations`](#integrations): Integrations linked to the workflow
- [`repository`](#repository): The repository linked to the workflow
- [`commit-status`](#commit-status): Commit status created by CDS on workflow run
- [`stages`](#stages): List of stages
- `env`: Define environment variable for the whole workflow
- [`gates`](#gates): Manual gate for your workflow
- [`from`](#from): Use a workflow template to generate this workflow
- [`parameters`](#parameters): Parameters input to generate workflow from referenced workflow template
- [`retention`](#retention): Workflow run retention in days. It override the workflow retention set on the project

<span style="color:red">\*</span> mandatory fields

## Repository

The repository linked to you workflow allows you to:

- Listen event to trigger the workflow through the field [`on`](#on)
- Use actions as `checkout` that simply git clone action

## Commit-status:

A commit build status is always sent by CDS with default values. You can customize the title and description of the build status with the `commit-status` attribute.

```yaml
commit-status:
  title: foo
  description: bar
```

## On

Available hooks:

- `push`: trigger the workflow on repository push event
- `pull-request`: trigger the workflow on repository pull-request event, see types of pull-request below.
- `model-update`: trigger the workflow is a worker model used in the worker has been updated
- `workflow-update`: trigger the workflow is the workflow definition was updated

`model-update` and `workflow-update` are only available is the workflow definition is different from the `repository` field of your workflow. The hook will be triggered when default branch is updated, and will trigger the default branch of the destination repository

the `on` field has 2 formats

### Array of string:

```yaml
on: [push, pull-request, model-update, workflow-update]
```

### Map

```yaml
on:
  push:
    branches: [main, develop]
    paths: [^src/.*/.*.java$]
  model-update:
    models: [MYPROJ/github/ovh/resources/mymodel]
    target_branch: main
  pull-request:
    comment: "a comment here"
    types: ["opened", "reopened", "closed", "edited"]
    branches: [main, develop]
    paths: [^src/.*/.*.java$]
  pull-request-comment:
    comment: "a comment here"
    types: ["created", "deleted", "edited"]
    branches: [main, develop]
    paths: [^src/.*/.*.java$]
  workflow-update:
    target_branch: main
```

- `push.branches`: branches filter
- `push.paths`: file paths filter
- `pull-request.comment`: comment written by cds at workflow end if it was triggered by a pull-request event.
- `pull-request.types`: types of pull-request event that can trigger the workflow. Could be: `opened`, `reopened`, `closed`, `edited`.
- `pull-request.branches`: branches filter
- `pull-request.paths`: file paths filter
- `pull-request-comment.comment`: comment written by cds at workflow end if it was triggered by a pull-request-comment event.
- `pull-request-comment.types`: types of pull-request-comment event that can trigger the workflow. Could be: `created`, `deleted`, `edited`.
- `pull-request-comment.branches`: branches filter
- `pull-request-comment.paths`: file paths filter
- `model-update.models`: worker model filter
- `model-update.target_branch`: destination repository branch to trigger
- `workflow-update.target_branch`: destination repository branch to trigger

## Integrations

Allow a job to use an project integration.

Integration can be put directly on a [job](#jobs) or at the [workflow top level](#fields) to be applied on all jobs

Supported integrations:

- [Artifactory]({{< relref "/docs/integrations/artifact-manager.md" >}})

## jobs

Jobs field is a map that contains all the jobs of your workflow. The key of the map is the name that will be display in CDS UI

```yaml
jobs:
  myJob:
    runs-on: ./cds/worker-models/my-custom-ubuntu.yml
    vars: [varset1, varset2]
    integrations: [my-artifactory]
    steps:
      run: echo 'Hello World'
  mySecondJob:
    runs-on: ./cds/worker-models/my-custom-ubuntu.yml
    needs: [myJob]
    steps:
      run: echo 'Bye'
```

- <span style="color:red">\*</span>[`runs-on`](#runs-on): define on which worker model your job will be executed
- <span style="color:red">\*</span>[`steps`](#step): the list of step to execute
- `name`: job description
- `needs`: the list of jobs that need to be executed before this one
- [`vars`](/docs/concepts/cds_as_code/project/variableset/): the list of variable set available in the job
- [`integrations`](#integrations): integration linked to the job
- `region`: the region on which the job must be triggered
- [`if`](#conditions): condition that must be satisfied to run the job. `if` and `gate` field cannot be set together
- `gate`: manual [gate](#gates) definition to use.`if` and `gate` field cannot be set together
- [`inputs`](#inputs): input of the job. If used, only these inputs can be used in the job steps. All others contexts cannot be used
- `stage`: link the job to a [stage](#stage)
- `continue-on-error`: if `true`, the job will be considered as Success when it fails
- `integrations`: link [project integrations](/docs/integrations/) to your job. Available integration: `artifactory`
- [`strategy`](#strategy): add a run strategy
- [`services`](#services): add container services to run with your job.
- `env`: define environment variables to inject to your job. It overrides environment variable with the same name defined at the workflow level

### Runs-On

Runs-on represents the worker model that will be used to execute your job.

If you don't need any customization, you can write it like this:

```yaml
runs-on: ./cds/worker-models/my-custom-ubuntu.yml
```

If you need to adjust memory used by the worker or the flavor to use:

```yaml
runs-on:
  model: ./cds/worker-models/my-custom-ubuntu.yml
  flavor: b2-7 # Only for openstack model
  memory: 4096 # Only for docker model
```

You can adjust the me

### Step

A step represent

```yaml
jobs:
  myjob:
    steps:
      - id: stepIdentifier
        run: echo 'Hello World' # cannot be used with `uses`
        uses: actions/checkout # cannot be used with `run`
        with:
          ref: develop
          sha: aefd1235
        if: failure()
        continue-on-error: true
        env:
          NEW_VAR: myValue
```

- `id`: step identifier
- `run`: script to execute. Cannot be used simultaneously with `uses` field
- `uses`: action to execute. Cannot be used simultaneously with `run` field
- `with`: allow you to customize action input. Must be used with `uses` field
- [`if`](#conditions): condition that must be satisfied to execute the step
- `continue-on-error`: if `true`, the step will be considered as Success when it fails
- `env`: define environment variables to inject to your job. It overrides environment variable with the same name defined oat the workflow and job level

### Inputs

Inputs allow you to define a list of variable that will be used in your job. If you use it all others contexts will be unavailable. This allows you to exactly control the inputs of your job

```yaml
jobs:
  myjob:
    inputs:
      inp1: ${{ git.ref }}
      inp2: ${{ cds.workflow }}
      inp3: My Value
```

### Strategy

Allow you to define a execution strategy for your job.

Available strategy:

- matrix

```yaml
jobs:
  myjob:
    strategy:
      matrix:
        version: ["go1.21", go1.22]
        os: [ubuntu, debian]
    steps:
      run: echo ${{ matrix.version }} - ${{ matrix.os }}
```

The matrix strategy allows you to template a job with matrix variables that will automatically create multiple jobs during the execution

In this example, CDS will create 4 jobs during execution with the given matrix context:

- job1: matrix.Version = go1.21 / matrix.os = ubuntu
- job2: matrix.Version = go1.21 / matrix.os = debian
- job3: matrix.Version = go1.22 / matrix.os = ubuntu
- job4: matrix.Version = go1.22 / matrix.os = debian

### Services

Service are docker containers spawned with your job in a private network. For example it allows you to start a postreSQL DB for your tests

```yaml
jobs:
  init:
    runs-on: .cds/worker-models/buildpack-deps-buster.yml
    services:
      myngnix:
        image: nginx:1.13
        env:
          NGINX_PORT: 80
        readiness:
          command: curl --fail http://myngnix:80
          interval: 10s
          timeout: 5s
          retries: 5
      mypostgres: ...
```

- `image`: The docker image of the service
- `env`: Environment variable to inject in the service
- `readiness`: Allows you to configure a readiness test for your service. Your job will wait for it before starting the steps execution
  - `command`: Command to execute to check the readiness of the service
  - `interval`: Interval between 2 tests
  - `timeout`: Command timeout before failing
  - `retries`: Number of retries

## Gates

Gates are hooks that allow you to manually trigger a job under certain conditions

```yaml
gates:
  first-gate:
    if: ${{ git.ref == 'main' && gate.approve }}
    inputs:
      approve:
        type: boolean
    reviewers:
      groups: [release-team]
jobs:
  myGateJob:
    gate: first-gate
```

- [`if`](#conditions): condition that must be satisfied to pass the gate
- `inputs`:
  - `type`: type of the input (boolean, number, text)
- `reviewers`: Allow you to define who can trigger the gate
  - `groups`: list of groups that are allowed to trigger the gate
  - `users`: list of users that are allowed to trigger the gate

## Stages

The use of stages allows you to structure and organize jobs in a modular way

```yaml
stages:
  my-stage:
  my-stage2:
    needs: [my-stage]
```

- `needs`: the list of stages that need to be executed before this one

## From

Specifying `from` will allow your workflow to use a workflow template.
If set, all other fields will be generated from the template (except workflow name).

You can reference either locally from the same repository (e.g. `.cds/workflow-templates/...`) or its entity path (e.g. `PROJECT_KEY/vcs/repository/name`).

```yaml
from: ".cds/workflow-templates/template.yaml"
```

## Parameters

Specify inputs to use with referenced workflow template with `from`.
It expects a map with string keys and string values.

```yaml
parameters:
  var1: "value1"
  var2: "value2"
```

# Conditions

Condition can be use at different level but share the same syntaxe

- workflow gate
- job.if
- step.if

To build your condition, you can use:

- [contexts](./../../contexts/)
- builtin functions:
  - `success()`: it will check if parent jobs / previous steps are in success
  - `failure()`: it will check if parent jobs / previous steps are in error
  - `always()`: it will always execute the job/step

You can use all [contexts](./../../contexts/) to create your condition

## Syntax

```
if: ${{ git.ref == "master" && cds.job == "MyJob" }}
or
if: git.ref == "master" && cds.job == "MyJob"
```

### Operators list

- `==`
- `!=`
- `>`
- `<`
- `>=`
- `<=`
- `||`
- `&&`
- `!`
