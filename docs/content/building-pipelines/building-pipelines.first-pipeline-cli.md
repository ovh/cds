+++
title = "First Pipeline with CLI"
weight = 2

[menu.main]
parent = "building-pipelines"
identifier = "first-pipeline-cli"

+++


### Create a project

The first step is to create your own Project.
A project contains applications, pipelines and environments.

You can use either the CLI or the UI to create your project.

```bash
$ cds project add TEST "My test project" test-team
OK
```

### Create a simple pipeline

```bash
$ cds pipeline add TEST hello-pip
Pipeline hello-pip created.
```

### Create an application

```bash
$ cds application add TEST hello-world
Aplication hello-world created.
```

### Configure your pipeline

We will add a script action in pipeline saying "Hello World !"

```bash
* cds pipeline job add DEMO hello-pip myJob1
$ cds pipeline job append DEMO hello-pip myJob1 Script -p script="echo Hello World! "
```

Last step, attach the pipeline to the application

```bash
$ cds application pipeline add DEMO hello-world hello-pip
OK
```

### Run your pipeline

You can test the pipeline providing the name parameter:
```bash
$ cds pipeline run DEMO hello-world hello-pip
DATE                       JOB-STEP                   LOG
seconds:1493973274         369749-0                   Starting step /Script-1
seconds:1493973274         369749-0                   Hello World!
seconds:1493973274         369749-0                   End of step /Script-1 [Success]
seconds:1493973278         0-0                        Build finished with status: Success
```
