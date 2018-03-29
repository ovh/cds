+++
title = "Workflow configuration files"
weight = 3

+++

Using `CDS Workflow Configuration Files`, checked into source controle, provides several benefits:
* Code reviews on the workflow configuration
* Ability to update workflow configuration from particular branch
* Gain efficienty while editing the workflow configuration

This usage of `CDS Workflow Configuration Files` is particulary suitable for single repository CI/CD workflows.

`CDS Workflow Configuration Files` relies on several text files on YAML syntax in a `.cds` directory at the root of your repository. Several files may be used to describe properly each component, such as applications, environments and pipelines of your CDS workflow.

## Basic example

This basic example represents a simple Java application which will be built and deployed from CDS. 

```
➜  cdsdemo git:(master) ls -la
total 8
drwxr-xr-x   6 fsamin  staff  192 Mar 27 11:01 .
drwxr-xr-x   3 fsamin  staff   96 Mar 27 11:01 ..
drwxr-xr-x   6 fsamin  staff  192 Mar 27 11:01 .cds
drwxr-xr-x  12 fsamin  staff  384 Mar 27 11:07 .git
-rw-r--r--   1 fsamin  staff   23 Mar 27 11:01 .gitignore
drwxr-xr-x   4 fsamin  staff  128 Mar 27 11:01 api

```

At the root of the repository a `.cds` directory contains all the `CDS Configuration Files`.

```
➜  cdsdemo git:(master) cd .cds
➜  .cds git:(master) ls -la
total 32
drwxr-xr-x  6 fsamin  staff  192 Mar 27 11:01 .
drwxr-xr-x  6 fsamin  staff  192 Mar 27 11:01 ..
-rw-r--r--  1 fsamin  staff  201 Mar 27 11:01 build.pip.yml
-rw-r--r--  1 fsamin  staff  104 Mar 27 11:01 deploy.pip.yml
-rw-r--r--  1 fsamin  staff   28 Mar 27 11:01 demo.app.yml
-rw-r--r--  1 fsamin  staff  178 Mar 27 11:01 democds.yml
```

Here we have 4 yaml files, two pipelines: `build.pip.yml` and `deploy.pip.yml`, one application `demo.app.yml` and the overall workflow description file `democds.yml`.

### Workflow syntax
First of all, here the workflow file:

```yaml
name: democds
workflow:
  build:
    pipeline: build-jar
    application: demo
  deploy:
    depends_on:
    - build
    pipeline: deploy-jar
    application: demo
```

This is the representation of the whole workflow, starting from `build` then `deploy`.  As shown, the workflow file only contains the description of pipelines orchestration.

Read more about CDS [workflow syntax]({{< relref "workflows/files/workflow-syntax.md" >}})

### Pipeline syntax
The pipelines files represents the most important part of your workflow. The pipeline file represents the jobs triggered within parallel stages.

Read more about CDS [pipeline syntax]({{< relref "workflows/files/pipeline-syntax.md" >}})

### Application syntax
The application file describe the application and the way to checkout it. It can also set number of variables.

Read more about CDS [application syntax]({{< relref "workflows/files/application-syntax.md" >}})

### Environment syntax
You can attach an environment to a pipeline in a workflow. An environemnt is basically a set of variables.

Read more about CDS [environment syntax]({{< relref "workflows/files/environment-syntax.md" >}})




