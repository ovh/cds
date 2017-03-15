# CDS Pipeline configuration file

## Pipeline Concepts

A pipeline describes how things need to be executed in order to achieve wanted result. In CDS, a pipeline a defined on a project and can be used on several applications inside the same project.

A pipeline is structured in sequential **stages** containing one or multiple concurrent **jobs**.

In CDS there is several types of pipeline : **build**, **testing** and **deployment**. In Pipeline configuration file, default type is **build**.

The goal is to make your pipeline the more reusable as possible. It have to be able to build, test or deploy all the tiers, services or micro-services of your project.

You can also define ACL on a pipeline.

### Stages

Usually in CDS a **build** pipeline is structured of the following stages :

- **Compile stage** : Build the binaries
- **Analysis & Unit Tests stage** : Run all unit tests and analyse code quality
- **Packaging stage** : Build the final package, Virtual Machine Image or Docker Image.

In CDS, stages are executed sequentially if the previous stage is successfull.

You can define trigger conditions on a stage, to enable it on certain conditions. For instance, you want to run the *Compile Stage* and *Analysis & Unit Tests stage* on all branches but the *Packaging Stage* on branches master and develop only.

A **Stage** is a set of jobs wich will be run in parallel.

### Jobs

The Job is more important concept in CDS. It will be composed of **steps** which will be run sequencially. A Job will be executed is a dedicated workspace and each new run of a job will have a new dedicated workspace. It means that you cannot share a workspace between jobs or between two runs of a job.

A Job will be executed by a **worker**. CDS will choose and provision a worker for dependending of the **requirements** you define on your job.
You can set as many requirements as you want, following those rules :

- Only one model can be set as requirement
- Only one hostname can be set as requirement
- Memory and Services requirements are availabe only on Docker models

If you want to share files or artifact between jobs, stages or pipelines you have to use *Artifact upload* and *Artifact download*. You can also share variable between stages, see [variables tutorial](variables.md) for more details.

### Steps

The steps of a job is the list of the different operation performed by the CDS worker. Each steps is based on an **Action** which is defined by CDS. The list of all actions is defined on *<your cds url ui>/#/action*. On the very first step failed, the job is marked as Failed and execution is stopped.

You can define a Step as final. It mean that even if the job is failed, the step will be executed. The *final* steps are executed after all other steps.

## How to write a configuration file

You can define a pipeline both in json format or in yaml format. Default is yaml format.

### Basic configuration

If you have a pretty simple *build* pipeline with one stage and one job. You can write such a configuration file

```yaml
steps:
- script: echo I'm the firt step
- script: echo I'm the second step
```

This defines a pipeline of type `build` (it's the default type), nammed `Build` (the default name for a build Pipeline). It will have a Stage nammed **Build** (the default stage name for a one stage pipeline is the name of the pipeline), with a job **Build** (the default job name for a one job stage is the name of the stage) composed of thow steps using script actions.

It is basically equivalent as :

```yaml
name: Build
type: build
stages:
  1|Build:
    jobs:
      Build:
        steps:
        - script: echo I'm the firt step
        - script: echo I'm the second step
```

A bit more complex example with two jobs, with requirements and other kind of steps :

```yaml
name: maven-build
jobs:

  Compile:
    requirements:
    - binary: mvn

    steps:
    - GitClone:
        url: '{{.git.http_url}}'
        branch: '{{.git.branch}}'
        commit: '{{.git.hash}}'
        directory: .
    - script: mvn compile

  Unit Tests:
    requirements:
    - binary: mvn

    steps:
    - GitClone:
        url: '{{.git.http_url}}'
        branch: '{{.git.branch}}'
        commit: '{{.git.hash}}'
        directory: .
    - script: mvn test
    - jUnitReport: ./target/surefire-reports*.xml
```

### Advanced usage

Same use case as above, but we add a stage to build the package only on branch master and release

```yaml
name: maven-build

stages:
  1|Application Build:
    jobs:

      Compile:
        steps:
        - GitClone:
            url: '{{.git.http_url}}'
            branch: '{{.git.branch}}'
            commit: '{{.git.hash}}'
            directory: .
        - script: mvn compile
        requirements:
        - binary: bash
        - binary: git
        - binary: mvn

      Unit Tests:
        steps:
        - GitClone:
            url: '{{.git.http_url}}'
            branch: '{{.git.branch}}'
            commit: '{{.git.hash}}'
            directory: .
        - script: mvn test
        - jUnitReport: ./target/surefire-reports*.xml
        requirements:
        - binary: mvn

  2|Application Package:

    conditions:
      git.branch: master|release

    jobs:

      Package:
        steps:
        - GitClone:
            url: '{{.git.http_url}}'
            branch: '{{.git.branch}}'
            commit: '{{.git.hash}}'
            directory: .
        - script: mvn compile
        - script: |-
            #!/bin/bash
            echo "--- Starting packaging ---"
            mvn package -DskipTests=true
        - artifactUpload:
            path: target/*.tar/.gz
            tag: '{{.cds.version}}'
```

## Pipeline configuration export

You can exported full configuration of your pipeline with the CDS CLI :

```bash
cds pipeline export PROJECT_KEY pipeline_name
```

Usage:

```bash
cds pipeline export <projectKey> <pipeline>

Usage:
  cds pipeline export [flags]

Flags:
      --format string     Format: json|yaml|hcl (default "yaml")
      --output string     Output filename
      --withPermissions   Export pipeline configuration with permission

Global Flags:
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

## Pipeline configuration import

You can import full configuration of your pipeline with the CDS CLI :


```bash
cds pipeline import PROJECT_KEY pipelinefile
```

Usage:

```bash
See documentation on https://github.com/ovh/cds/tree/master/doc/tutorials

Usage:
  cds pipeline import [flags]

Flags:
      --format string   Configuration file format (default "yaml")
      --url string      Import pipeline from an URL

Global Flags:
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output

```
