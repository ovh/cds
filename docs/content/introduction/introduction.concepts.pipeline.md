+++
title = "Pipeline"
weight = 1

[menu.main]
parent = "concepts"
identifier = "concepts-pipeline"

+++

![Pipeline](/images/concepts_pipeline.png)

A pipeline describes how things need to be executed in order to obtain the expected result. In CDS, a pipeline belongs to a single project and can be used with the applications of that project.

A pipeline is structured in sequential **[stages]({{< relref "introduction.concepts.stage.md" >}})** containing one or multiple concurrent **[jobs]({{< relref "introduction.concepts.job.md" >}})**.

In CDS there are several types of pipelines : **build**, **testing** and **deployment**. In Pipeline configuration file, default type is **build**.

CDS pipelines can be parametrized. This allows you to reuse the same pipeline when you have similar workloads. For example, you could use the same pipeline to deploy in your pre-production environment first and then to your production environment.

You can also define ACL on a pipeline.

## Triggers

![Triggers](/images/concepts_pipeline_trigger.png)

## Example

![Example](/images/concepts_pipeline_example.png)
