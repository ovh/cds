---
title: "Variables"
weight: 6
tags: ["variable", "variables", "helper", "helpers", "interpolate"]
card: 
  name: concept_pipeline
  weight: 4
---

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
- `{{.cds.project}}` The key of the current project
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
- `{{.cds.workspace}}` Current job's workspace. It's a directory. In a step [script]({{< relref "/docs/actions/builtin-script.md" >}}), `{{.cds.workspace}}` == $HOME
- `{{.payload}}` The current payload in JSON format

## The cds.version variable

`{{.cds.version}}`

CDS version is a builtin variable, it is transmitted through pipelines of a workflow run.

## Export a variable inside a step

In a step of type `script`, you can export a variable as the following:

```bash
$ worker export varname thevalue
```

You can use the build variable in:

* the current job with `{{.cds.build.varname}}`
* the next stages in same pipeline `{{.cds.build.varname}}`
* the next pipelines `{{.workflow.pipelineName.build.varname}}` with `pipelineName` the name of the pipeline in your workflow

[See worker export documentation]({{< relref "/docs/components/worker/export.md" >}})

## Shell Environment Variable

All CDS variables, except `password type`, can be used as plain environment variables.

Theses lines will have the same output

```bash
echo '{{.cds.parent.application}}'
echo $CDS_PARENT_APPLICATION
```

## Git variables

Here is the list of git variables:

- `{{.git.hash.before}}`: SHA of the most recent commit before the push
- `{{.git.hash}}`: SHA of the most recent commit after the push
- `{{.git.hash.short}}`: Short version of git.hash
- `{{.git.hook}}`: Name of the event that trigger the run
- `{{.git.url}}`:  Git ssh URL used to clone
- `{{.git.http_url}}`: Git http url used to clone
- `{{.git.branch}}`: 
  - Push event: Name of the branch where the push occured
  - PullRequest event: Name of the source branch
- `{{.git.tag}}`: Name of the tag that triggered the run
- `{{.git.author}}`: Name of the most recent commit author
- `{{.git.author.email}}`: Email of the most recent commit author
- `{{.git.message}}`: Git message of the most recent commit
- `{{.git.server}}`: Name of the repository manager
- `{{.git.repository}}`: 
  - Push event:  Name of the repository
  - PullRequest event: Name of the source repository

Here is the list of git variables available only for Bitbucket server

- `{{.git.hash.dest}}`: SHA of the most rcent commit on destination branch ( PullRequest event )
- `{{.git.branch.dest}}`: Name of the destination branch on a pull request event
- `{{.git.repository.dest}}`: Name of the target repository on a pull request event
- `{{.git.pr.id}}`: Identifier of the pullrequest
- `{{.git.pr.title}}`: Title of the pullrequest
- `{{.git.pr.state}}`: Status of the pullrequest
- `{{.git.pr.previous.title}}`: Previous title of the pullrequest
- `{{.git.pr.previous.branch}}`: Previous target branch of the pullrequest
- `{{.git.pr.previous.hash}}`: Previous target hash of the pullrequest
- `{{.git.pr.previous.state}}`: Previous status of the pullrequest
- `{{.git.pr.reviewer}}`: Name of the reviewer
- `{{.git.pr.reviewer.email}}`: Email of the reviewer
- `{{.git.pr.reviewer.status}}`: Status of the review
- `{{.git.pr.reviewer.role}}`: Role of the reviewer
- `{{.git.pr.comment}}`: Comment written by the reviewer
- `{{.git.pr.comment.before}}`: Previous comment
- `{{.git.pr.comment.author}}`: Author name of the comment
- `{{.git.pr.comment.author.email}}` Author email of the comment



## Pipeline parameters

On a pipeline, you can add some parameters, this will let you to use `{{.cds.pip.varname}}` in your pipeline. 
Then, in the workflow, you can set the value for pipeline parameter in the `pipeline context`.

Notice that you can't create a pipeline parameter of type `password`. If you want to use a variable of type password, you have to create it in your project / application or environment. Then, in your workflow, use this variable to set the value of the pipeline parameter - the pipeline parameter can be of type `string`.


## Helpers

Some helpers are available to transform the value of a CDS Variable.

Example: run a pipeline, with an application named `my_app`. A step script:

```
echo "{{.cds.application | upper}}"
```

will display

```
MY_APP
```

Helpers available and some examples:

- abbrev
- abbrevboth
- trunc
- trim
- upper: `{{.cds.application | upper}}`
- lower: `{{.cds.application | lower}}`
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
- replace: `{{.cds.application | replace "_" "."}}`
- plural
- toString
- default: `{{.cds.application | default ""}}`, `{{.cds.application | default "defaultValue"}}`, `{{.cds.app.foo | default .cds.app.bar .cds.app.biz }}`
- empty
- coalesce
- toJSON
- toPrettyJSON
- b64enc
- b64dec
- escape: replace '_', '/', '.' by '-'

### Advanced usage

You can use CDS Variables with default helpers:

```
{{.cds.app.foo | default .cds.app.bar }}
```

You can use many helpers:

```
{{.cds.app.foo | upper | lower}}
{{.cds.app.foo | default .cds.app.bar | default .cds.app.biz | upper }}
```

### Deep in code

Are you a Go developer? See all helpers on https://github.com/ovh/cds/blob/master/sdk/interpolate/interpolate_helper.go#L23
and some unit tests on https://github.com/ovh/cds/blob/master/sdk/interpolate/interpolate_test.go#L72 
