---
title: "Workflow"
weight: 7
card: 
  name: concept_workflow
  weight: 1
---

The workflow concept is a key feature, and widely considered a native, manageable and feature-rich entity in CDS. A CDS Workflow allows you to chain pipelines with manual or automatic gates, using conditional branching. A workflow can be stored as code, designed on CDS UI, or both, depending on what best suits you. 

Let’s take an example. One workflow for building and deploying three micro-services:  

- Build each micro-service 
- Deploy them in preproduction 
- Run integration tests on preproduction environment 
- Deploy them in production, then re-run integration tests in production

![Workflow](./images/workflow.png?width=1000px)

For the building part, there is only one pipeline to manage, which is used three times in the workflow with a different application/environment context each time. This is called the [pipeline context]({{< relref "/docs/concepts/workflow/pipeline-context.md" >}}). 

Any conditional branching against the workflow (e.g.“automatic deployment on the staging environment, only if the current Git branch is master”) can be executed through a [run conditional]({{< relref "/docs/concepts/workflow/run-conditions.md" >}}) set on the pipeline. 

![Run Conditions](./images/run_conditions.png?width=600px)

Let’s look at a  a real use case. This is the workflow that builds, tests and deploys CDS in production at OVH (yes, CDS builds and deploys itself!):

![CDS Workflow](./images/workflow_cds.png?width=1000px)

1. For each Git commit, the workflow is triggered 
1. The UI is packaged, all binaries are prepared, and the docker images are built. The “UT” job launches the unit tests. The job “IT” job installs CDS in an ephemeral environment and launches the integration tests on it. Part 2 is automatically triggered on all Git commits.  
1. Part 3 deploys CDS on our preproduction environment, then launches the integration tests on it. It is started automatically when the current branch is the master branch. 
1. Last but not least, part 4 deploys CDS on our production environment. 

If there is a failure on a pipeline, it may look like this:

![CDS Workflow Failure](./images/workflow_cds_failure.png?width=600px)

But of course, you’re not limited to the most complex tasks with CDS Workflows! These two examples demonstrate the fact that workflows allow to build and deploy a coherent set of micro-services. If you have simpler needs, your workflows are, of course, simpler.


![CDS Workflow Failure](./images/workflow_simple.png?width=300px)

