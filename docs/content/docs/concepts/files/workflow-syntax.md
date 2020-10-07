---
title: "Workflow configuration file"
weight: 1
card: 
  name: concept_workflow
  weight: 2
---

A CDS workflow file only contains the description of pipelines orchestration, hooks, run conditions, etc. 
Consider the following workflow wich implements a basic two-stage workflow:

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
    one_at_a_time: true
hooks:
  build:
  - type: RepositoryWebHook
notifications:
- type: email
  pipelines:
  - deploy
  settings:
    on_success: never
    recipients:
    - me@foo.bar
retention_policy: return run_days_before < 7
```

There are two major things to understand: `workflow` and `hooks`. A workflow is a kind of graph starting from a root pipeline, and other pipelines with dependencies. In this example, the `deploy` pipeline will be triggered after the `build` pipeline.

## Run Conditions
[Run Conditions documentation]({{<relref "/docs/concepts/workflow/run-conditions.md">}})

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

## Notifications

[Notifications documentation]({{<relref "/docs/concepts/workflow/notifications.md">}})

Example of email notification.

```yml
- type: email
  pipelines:
  - deploy
  settings:
    on_success: never
    recipients:
    - me@foo.bar
```

Example of jabber notification. Note that you can add `conditions` on every notifications

```yml
- type: jabber
  pipelines:
  - deploy
  settings:
    on_start: true
    send_to_groups: true
    recipients:
    - me@jabber.com
    conditions:
      check:
      - variable: cds.triggered_by.email
        operator: eq
        value: me@localhost.local
```

Example of vcs notification. Note that `pipelines` list is optional on every notifications. When it's not specified, notification will be triggered for each pipeline

```yml
- type: vcs
  settings:
    template:
      body: |+
        [[- if .Stages ]]
        CDS Report [[.WorkflowNodeName]]#[[.Number]].[[.SubNumber]] [[ if eq .Status "Success" -]] ✔ [[ else ]][[ if eq .Status "Fail" -]] ✘ [[ else ]][[ if eq .Status "Stopped" -]] ■ [[ else ]]- [[ end ]] [[ end ]] [[ end ]]
        [[- end]]
      disable_comment: false
      disable_status: false
```

## Mutex

[Mutex documentation]({{<relref "/docs/concepts/workflow/mutex.md">}})

Example of a pipeline limited to one execution at a time: deployments to production cannot be executed concurently.

```yml
name: my-workflow
workflow:
  # ...
  deploy:
    pipeline: deploy
    # ...
    one_at_a_time: true # No concurent deployments
```

## Retention Policy

[Retention documentation]({{<relref "/docs/concepts/workflow/retention.md">}})
