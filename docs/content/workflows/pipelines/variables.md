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

- `{{.cds.environment}}` The name of the current environment
- `{{.cds.application}}` The name of the current application
- `{{.cds.job}}` The name of the current job
- `{{.cds.manual}}` true if current pipeline is manually run, false otherwise
- `{{.cds.pipeline}}` The name of the current pipeline 
- `{{.cds.project}}` The name of the current project
- `{{.cds.run}}` Run Number of current workflow, example: 3.0
- `{{.cds.run.number}}` Number of current workflow, example: 3 if `{{.cds.run}} = 3.0`
- `{{.cds.run.subnumber}}` Sub Number of current workflow, example: 4 if `{{.cds.run}} = 3.4`
- `{{.cds.stage}}` The name of the current stage
- `{{.cds.status}}` Status or previous pipeline: Success or Failed
- `{{.cds.triggered_by.email}}` Email of the user who launched the current build
- `{{.cds.triggered_by.fullname}}` Full name of the user who launched the current build
- `{{.cds.triggered_by.username}}` Username of the user who launched the current build
- `{{.cds.version}}` The current version number, it's an alias to `{{.cds.run.number}}`
- `{{.cds.workflow}}` The name of the current workflow
- `{{.cds.workspace}}` Current job's workspace. It's a directory. In a step [script]({{< relref "workflows/pipelines/actions/builtin/script.md" >}}), `{{.cds.workspace}}` == $HOME

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

## Helpers

Some helpers are available to transform the value of a CDS Variable.

Example: run a pipeline, with an application named `my_app`. A step script:

```
echo "{{.cds.application | upper}}
```

will display

```
MY_APP
```

Helpers available:

- abbrev
- abbrevboth
- trunc
- trim
- upper
- lower
- title
- untitle
- substr
- repeat
- trimall
- trimAll
- trimSuffix
- trimPrefix
- nospace
- initials
- randAlphaNum
- randAlpha
- randASCII
- randNumeric
- swapcase
- shuffle
- snakecase
- camelcase
- quote
- squote
- indent
- nindent
- replace
- plural
- toString
- default
- empty
- coalesce
- toJSON
- toPrettyJSON
- b64enc
- b64dec
- escape : replace '_', '/', '.' by '-'

You're a go developper? See all helpers on https://github.com/ovh/cds/blob/master/sdk/interpolate/interpolate_helper.go#L23 