---
title: "Pipeline"
weight: 3
card: 
  name: pipeline
---

![Pipeline](../images/pipeline.png?width=500px)

A pipeline describes how things need to be executed in order to obtain the expected result. In CDS, a pipeline belongs to a single project and can be used with the applications of that project.

A pipeline is structured in sequential **[stages]({{< relref "stage.md" >}})** containing one or multiple concurrent **[jobs]({{< relref "/docs/concepts/job.md" >}})**.

CDS pipelines can be parametrized. This allows you to reuse the same pipeline when you have similar workloads. For example, you could use the same pipeline to deploy in your pre-production environment first and then to your production environment.


A [Stage]({{ <relref "/docs/concepts/stage.md">}}) is a set of jobs that will be run in parallel. Stages are executed sequentially, if the previous stage is successful. 

Let’s take a real-life use case: the pipeline that built CDS. This pipeline has four stages: 

![Pipeline](../images/pipeline_cds.png?width=500px)

- The **Build Minimal** stage is launched for all Git branches. The main goal of this stage is to compile the Linux version of CDS binaries. 
- The **Build other os/arch** stage is only launched on the master branch. This stage compiles all  binaries supported by the os/arch: linux, openbsd, freebsd, darwin, windows – 386, amd64 and arm.  
- The **Package** stage is launched for all Git branches. This stage prepares the docker image and Debian package.
- Finally, the **Publish** stage is launched, whatever the Git branch. 

Most tasks are executed in parallel, whenever possible. This results in very fast feedback, so we will quickly know if the compilation is OK or not. 
