## Variable scopes

In CDS, it is possible to defines variables at different levels:

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

All variables in CDS can be invoked by using the simple “{{.VAR}}” format. To simplify usage between all variable sources, we have defined the following prefixes:

- Action variable: “{{.VAR}}”
- Builtin CDS: “{{.cds.VAR}}”
- Git hook: “{{.git.VAR}}”
- Pipeline: “{{.cds.pip.VAR}}”
- Application: “{{.cds.app.VAR}}”
- Environment: “{{.cds.env.VAR}}”
- Project: “{{.cds.proj.VAR}}”

## Builtin variables

Here is the list of builtin variables, generated for every build:

- ”{{.cds.pipeline}}” The name of the current pipeline
- ”{{.cds.application}}” The name of the current application
- ”{{.cds.project}}” The name of the current project
- ”{{.cds.environment}}” The name of the current environment
- ”{{.cds.version}}” The number of the current version
- ”{{.cds.parent.application}}” The name of the application that triggered the current build
- ”{{.cds.parent.pipeline}}” The name of the pipeline that triggered the current build
- ”{{.cds.triggered_by.email}}” Email of the user that run the current build
- ”{{.cds.triggered_by.fullname}}” Full name of the user that run the current build
- ”{{.cds.triggered_by.username}}” User that run the current build

## The .version variable

CDS version is a builtin variable equals to the buildNumber of the last pipeline of type “build”. This variable is transmitted through triggers with the same value to testing and deployment pipelines.

## Git variables

Here is the list of git variables:

- ”{{.git.hash}}”
- ”{{.git.branch}}”
- ”{{.git.author}}”
- ”{{.git.project}}”
- ”{{.git.repository}}”
