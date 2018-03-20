+++
title = "Use Configuration File"
weight = 4

+++


You can define a pipeline in the json format but also in the yaml format. The default is the yaml format.

### Basic configuration

If you have a pretty basic *build* pipeline with a single stage and a single job. You may write such a configuration file:

```yaml
steps:
- script: echo I'm the firt step
- script: echo I'm the second step
```

This defines a pipeline of type `build` (it's the default type), named `Build` (the default name for a build Pipeline). It will have a Stage named **Build** (the default stage name for a one stage pipeline is the name of the pipeline), with a job **Build** (the default job name for a one job stage is the name of the stage) composed of two steps using [script]({{< relref "workflows/pipelines/actions/builtin/script.md" >}}) actions.

It is basically equivalent to :

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

Here is a slightly more complex example with two jobs, with requirements and other kinds of steps :

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

Here is a config file with the same use case as above, but it adds a stage to build the package only on `master` and `release` branches.

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

You can export a full configuration of your pipeline with the CDS CLI using the `export` subcommand:

```bash
cdsctl pipeline export PROJECT_KEY pipeline_name
```

Usage:

```bash
Export CDS pipeline

Usage:
  cdsctl pipeline export [ PROJECT-KEY ] PIPELINE-NAME [flags]

Flags:
      --format string     yml or json (default "yml")
  -h, --help              help for export
      --with-permission   true or false

Global Flags:
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

## Pipeline configuration import

You can import a full configuration of your pipeline with the CDS CLI using the `import` subcommand:

```bash
cdsctl pipeline import PROJECT_KEY pipelinefile
```

Usage:

```bash
PATH: Path or URL of pipeline to import 

Usage:
  cdsctl pipeline import [ PROJECT-KEY ] PATH [flags]

Flags:
      --force   Use force flag to update your pipeline
  -h, --help    help for import

Global Flags:
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```
