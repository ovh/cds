---
title: "Pipeline"
weight: 3
card: 
  name: concept_pipeline
  weight: 1
---

![Pipeline](../images/pipeline.png?width=500px)

A pipeline describes how things need to be executed in order to obtain the expected result. In CDS, a pipeline belongs to a single project and can be used with the applications of that project.

A pipeline is structured in sequential **stages** containing one or multiple concurrent **[jobs]({{< relref "/docs/concepts/job.md" >}})**.

CDS pipelines can be parametrized. This allows you to reuse the same pipeline when you have similar workloads. For example, you could use the same pipeline to deploy in your pre-production environment first and then to your production environment.

A **stage** is a set of jobs that will be run in parallel. Stages are executed sequentially, if the previous stage is successful. 

Let’s take a real-life use case: the pipeline that built CDS. This pipeline has four stages: 

![Pipeline](../images/pipeline_cds.png?width=500px)

- The **Build Minimal** stage is launched for all Git branches. The main goal of this stage is to compile the Linux version of CDS binaries. 
- The **Build other os/arch** stage is only launched on the master branch. This stage compiles all  binaries supported by the os/arch: linux, openbsd, freebsd, darwin, windows – 386, amd64 and arm.  
- The **Package** stage is launched for all Git branches. This stage prepares the docker image and Debian package.
- Finally, the **Publish** stage is launched, whatever the Git branch. 

Most tasks are executed in parallel, whenever possible. This results in very fast feedback, so we will quickly know if the compilation is OK or not. 

## Stages

Usually in CDS a **build** pipeline is structured of the following stages:

- **Compile stage**: Build the binaries
- **Analysis & Unit Tests stage**: Run all unit tests and analyse code quality
- **Packaging stage**: Build the final package, Virtual Machine Image or Docker Image.

A  Stage is a set of jobs that will be run in parallel. Stages are executed sequentially, if the previous stage is successful. 

You can define trigger conditions on a stage, to enable/disable it under given conditions. For instance, you may want to run the *Compile Stage* and *Analysis & Unit Tests stage* on all branches but dedicate the *Packaging Stage* run on `master` and `develop` branches only.

A **Stage** is a set of jobs which will be run in parallel.