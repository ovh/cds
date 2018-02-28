+++
title = "Concepts"
weight = 1

+++

### A Step

This is a step, using an action script. [See step concept]({{< relref "job.md" >}}) and [actions documentation]({{< relref "workflows/pipelines/actions/_index.md" >}}).

![Step](/images/introduction.concept.step.png)

### A Job

A job is composed of several steps. [See job concept]({{< relref "job.md" >}}).

![Step](/images/introduction.concept.job.png)


### A Stage

A stage is composed of several jobs. [See stage concept]({{< relref "stage.md" >}}).

![Stage](/images/introduction.concept.stage.png)

### A Pipeline

A stage is composed of several jobs. [See pipeline concept]({{< relref "pipeline.md" >}}).

![Pipeline](/images/introduction.concept.pipeline.png)

### A Workflow

A stage is composed of several pipelines. [See Workflow documentation]({{< relref "workflows/design/_index.md" >}}).

![Workflow](/images/introduction.concept.workflow.png)

### A Project

A project represents an organisation, it contains workflows, applications, pipelines and environments. It can be linked to platforms

### A Platform

A platform represents a link between CDS and another external platform (Kafka, Openstack)

