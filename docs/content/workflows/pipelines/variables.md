+++
title = "Variables"
weight = 1

+++

In CDS, it is possible to define variables at different levels:

- Project
- Environment
- Application

## Variable types

Existing variable types:

- String
- Text
- Boolean
- Number
- Password
- Key

## Placeholder format

All variables in CDS can be invoked using the simple `{{.VAR}}` format. To simplify the use between all the variable sources, we have defined the following prefixes:

- Action variable: `{{.VAR}}`
- Builtin CDS: `{{.cds.VAR}}`
- Git: `{{.git.VAR}}`
- Pipeline: `{{.cds.pip.VAR}}`
- Application: `{{.cds.app.VAR}}`
- Environment: `{{.cds.env.VAR}}`
- Project: `{{.cds.proj.VAR}}`
- Exported variable at build time: `{{.cds.build.VAR}}`

## Builtin variables

Here is the list of builtin variables, generated for every build:

- `{{.cds.project}}` The name of the current project
- `{{.cds.environment}}` The name of the current environment
- `{{.cds.application}}` The name of the current application
- `{{.cds.pipeline}}` The name of the current pipeline
- `{{.cds.stage}}` The name of the current stage
- `{{.cds.job}}` The name of the current job
- `{{.cds.workspace}}` Current job's workspace. It's a directory. In a step [script]({{< relref "workflows/pipelines/actions/builtin/script.md" >}}), `{{.cds.workspace}}` == $HOME
- `{{.cds.version}}` The current version number
- `{{.cds.parent.application}}` The name of the application that triggered the current build
- `{{.cds.parent.pipeline}}` The name of the pipeline that triggered the current build
- `{{.cds.triggered_by.email}}` Email of the user who launched the current build
- `{{.cds.triggered_by.fullname}}` Full name of the user who launched the current build
- `{{.cds.triggered_by.username}}` Username of the user who launched the current build

## The .version variable

`{{.cds.version}}`

CDS version is a builtin variable set to the buildNumber of the last pipeline of type “build”. This variable is transmitted through triggers with the same value to both testing and deployment pipelines.

## Export a variable inside a step

In a step of type `script`, you can export a variable as the following:

```bash
$ worker export varname thevalue
```

You can now use `{{.cds.build.varname}}` in further steps and stages.

## Shell Environment Variable

All CDS variables, except `password type`, can be used as plain environment variables.

Theses lines will have the same output

```bash
echo '{{.cds.parent.application}}'
echo $CDS_PARENT_APPLICATION
```

## Git variables

Here is the list of git variables:

- `{{.git.hash}}`
- `{{.git.url}}`
- `{{.git.http_url}}`
- `{{.git.branch}}`
- `{{.git.author}}`
- `{{.git.message}}`
