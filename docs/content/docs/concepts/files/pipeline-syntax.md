---
title: "Pipeline configuration file"
weight: 2
card: 
  name: concept_pipeline
  weight: 2
---


A CDS workflow file only contains the description of pipelines orchestration, hooks, run conditions, etc. 
Consider the following Pipeline which implements a basic two-stage continuous delivery pipeline:

```yaml
version: v1.0
name: build

parameters:
  param_name:
    type: string
    default: default_value
    
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
    - echo {{.cds.pip.param_name}}  
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

This file describes three jobs (`Build UI`, `Test UI` and `Package UI`) in two stages `Compile` and `Package`. The two first jobs will be run in parallel in the first stage. When the first Stage is successful, the second stage containing the last job will be run.

A pipeline always begins with:

```yaml
version: v1.0
name: build
stages:
- Compile
- Package
```

where:

* `version: v1.0` represents the version of the syntax used in this file
* `name: build` is the name of the pipeline
* `stages` is the ordered list of the stages


## Jobs

Each job has several properties:

* **name** - `job: Build UI` defines the name as `Build UI`.
* **stage** - this is mandatory if you have more than one stage. It must be one of the list stages described above.
* **enabled** - can be omitted, true by default. If you want to disable a Job, set this property to false.
* **requirements** - the list of the requirements to match a worker. Read more about [requirements]({{< relref "/docs/concepts/requirement/_index.md" >}}).
* **steps** - the ordered list of steps.

## Steps

Each job is composed of steps. A step is an action performed by a [CDS Worker]({{< relref "/docs/components/worker/_index.md" >}}) within a workspace. Each step uses an [action]({{< relref "/docs/actions/_index.md" >}}) and the syntax is:

* if the action has only one parameter:

```yaml
- job: xxx
  steps:
  - myAction: myParameter
```

* if the action has more that one parameter:

```yaml
- job: xxx
  steps:
  - myAction: 
      myFistParameter: value
      mySecondParameter: value
```

Read more about available [actions]({{< relref "/docs/actions/_index.md" >}}).

### Optional 

It is possible to make a step optional. Even if this task fail the job will continue.


```yaml
- job: xxx
  steps:
  - myAction: 
      myFistParameter: value
      mySecondParameter: value
    optional: true
```

This also work for built-in action

```yaml
- job: xxx
  steps:
  - coverage:
      format: other
      path: "{{.cds.workspace}}/target/site/jacoco/jacoco.xml"
    optional: true
```
