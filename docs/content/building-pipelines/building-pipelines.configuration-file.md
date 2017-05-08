+++
title = "Use Configuration File"
weight = 3

[menu.main]
parent = "building-pipelines"
identifier = "first-pipeline-configuration-file"

+++


You can define a pipeline both in json format or in yaml format. Default is yaml format.

### Basic configuration

If you have a pretty simple *build* pipeline with one stage and one job. You can write such a configuration file

```yaml
steps:
- script: echo I'm the firt step
- script: echo I'm the second step
```

This defines a pipeline of type `build` (it's the default type), named `Build` (the default name for a build Pipeline). It will have a Stage named **Build** (the default stage name for a one stage pipeline is the name of the pipeline), with a job **Build** (the default job name for a one job stage is the name of the stage) composed of thow steps using script actions.

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
