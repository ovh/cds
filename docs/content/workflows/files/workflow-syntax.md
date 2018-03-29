+++
title = "Workflow syntax"
weight = 1

+++

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

TODO