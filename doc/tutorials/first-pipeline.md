## Introduction

This tutorial will guide through the setup of a functionnal building pipeline.

Have you created your account ? If no, see the [quickstart](/doc/overview/quickstart.md).


### Create a project

The first step is to create your own [project](/overview/project).
A project contains [applications](/overview/application), [pipelines](/overview/pipeline)
and [deployment environments](/overview/environment).

You can use either the cli or the ui to create your project.

```shell
$ cds project add TEST "My test project" test-team
OK
```

In case of conflict in project key, you will see
```shell
$ cds project add TEST "My test project" test-team
Error: cannot add project My test project (Conflict: please use another project key)
```

### Create a simple pipeline

```shell
$ cds pipeline add TEST hello-pip
Pipeline hello-pip created.
```

### Create an application 

```shell
$ cds application add TEST hello-world
Aplication hello-world created.
```

### Configure your pipeline

We will add a script action in pipeline saying "Hello <something> !"

```shell
$ cds pipeline action add TEST hello-pip Script -p script="echo Hello {{.cds.pip.name}}! "
Action Script added to pipeline hello-pip
```

Then we will create a pipeline parameter "name" without default value

```shell
$ cds pipeline parameter add TEST hello-pip name "" string "Name to be printed"
OK
```

Last step, attach the pipeline to the application and set "name" parameter to "World"

```shell
$ cds application pipeline add TEST hello-world hello-pip -p name="World"
OK
```

### Run your pipeline

You can test the pipeline providing the name parameter:
```shell
$ cds pipeline run TEST hello-world hello-pip
DATE                       ACTION                     LOG
2016-02-16 11:40:40        Script                     Hello World!
0001-01-01 00:00:00        SYSTEM                     Build finished with status: Success
```

### Cleanup

Delete everything with the cli
```shell
$ cds app delete TEST hello-world && cds pipeline delete TEST hello-pip && cds project delete TEST
OK
OK
OK
```

