---
title: "Workflow configuration file"
weight: 1
card: 
  name: concept_workflow
  weight: 2
---

A CDS workflow file only contains the description of pipelines orchestration, hooks, run conditions, etc. 
Consider the folowwing workflow wich implements a basic two-stages workflow

```yaml
name: my-workflow
workflow:
  build:
    pipeline: build
    application: my-application
  deploy:
    depends_on:
    - build
    when:
    - success
    pipeline: deploy
    application: my-application
    environment: my-production
hooks:
  build:
  - type: RepositoryWebHook
```

Here there are two major things to understand: `workflow` and `hooks`. A workflow is a kind of graph starting from a root pipeline, and other pipelines with dependencies. In this example, the `deploy` pipeline will be triggered after the `build` pipeline.

## Run Conditions

[Run Conditions documentation]({{ <relref "/docs/concepts/workflow/run-conditions.md">}})

Example of basic condition. Notice that the `when` attribute is optional, it's just a shortcut on condition `cds.status == Success`.

```yml
yourpipeline:
  depends_on:
  - theparentpipeline
  conditions:
    check:
    - variable: git.branch
      operator: ne
      value: master
  when:
  - success
```

Example with many checks:

```yml 
conditions:
  check:
  - variable: git.branch
    operator: eq
    value: master
  - variable: git.repository
    operator: eq
    value: ovh/cds
  when:
  - success
```

Example with using LUA syntax as advanced condition:

```lua
  conditions:
    script: return cds_manual == "true" or (cds_status == "Success" and git_branch
      == "master" and git_repository == "ovh/cds")
```
