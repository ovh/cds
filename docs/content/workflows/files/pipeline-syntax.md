+++
title = "Pipeline syntax"
weight = 2

+++


A CDS workflow file only contains the description of pipelines orchestration, hooks, run conditions, etc. 
Consider the following Pipeline which implements a basic two-stages continuous delivery pipeline.

```yaml
version: v1.0
name: build
stages:
- Compile
- Package


jobs:
- job: Build UI
  stage: Compile
  steps:
  - gitClone:
      branch: '{{.git.branch}}'
      commit: '{{.git.hash}}'
      directory: cds
      url: '{{.git.url}}'
  - script:
    - cd cds/ui
    - npm set registry https://registry.npmjs.org
    - npm install
    - ng build -prod --aot
    - tar cfz ui.tar.gz dist
  - artifactUpload:
      path: cds/ui/ui.tar.gz
      tag: '{{.cds.version}}'
  requirements:
  - binary: git
  - memory: "1024"
  - model: Node8.9.1

- job: Test UI
  stage: Compile
  enabled: false
  requirements:
  - binary: git
  - memory: "1024"
  - model: Node8.9.1

  steps:
  - gitClone:
      branch: '{{.cdsbuildgitbranch}}'
      commit: '{{.git.hash}}'
      directory: cds
      password: ""
      privateKey: ""
      url: '{{.cds.app.repo}}'
      user: ""
  - script:
    - export CHROME_BIN=chromium
    - npm set registry https://registry.npmjs.org
    - cd cds/ui
    - npm install
    - npm test

  - jUnitReport: ./cds/ui/tests/**/results.xml

- job: Package UI
  stage: Package
  requirements:
  - binary: docker
  steps:
  - artifactDownload:
      path: .

  - CDS_DockerPackage:
      dockerfileDirectory: .
      imageName: ovh/cds-ui
      imageTag: '{{.cds.version}}'
  
```

## Stages

This file describes three jobs (`Build UI`, `Test UI` and `Package UI`) in two stages `Compile` and `Package`. The two first jobs will be run in parallel in the first stage. When the first Stage will be successful, the second stage containing the last job will be run.

A pipeline will always begin with
```yaml
version: v1.0
name: build
stages:
- Compile
- Package
```

* `version: v1.0` represents the version of the syntax used in this file
* `name: build` is the name of the pipeline
* `stages` is the ordered list of the stages


## Jobs

Each job has several properties:

* **name** - `job: Build UI` defines the name as `Build UI`
* **stage** - this is mandatory if you have more than one stage. It must be one of the list stages described above
* **enabled** - can be omitted, true by default. If you want to disable a Job
* **requirements** - the list of the requirements to match a worker. Read more about [requirements]({{< relref "/workflows/pipelines/requirements/_index.md" >}})
* **steps** - the ordered list of steps 

## Steps

Each job is composed of steps. A step is an action performed by a [CDS Worker]({{< relref "/worker/_index.md" >}}) within a [workspace]{{< relref "/worker/workspace.md" >}}. Each step use an [action]({{< relref "/workflows/pipelines/actions/_index.md" >}}) and the syntax is:

If the action has only one parameter:

```yaml
- job: xxx
  steps:
  - myAction: myParameter
```

If the action has more that one parameter:

```yaml
- job: xxx
  steps:
  - myAction: 
      myFistParameter: value
      mySecondParameter: value
```

Read more about available [actions]({{< relref "/workflows/pipelines/actions/_index.md" >}})
