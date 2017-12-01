+++
title = "Pipeline"
weight = 1

[menu.main]
parent = "concepts"
identifier = "concepts-pipeline"

+++

![Pipeline](/images/concepts_pipeline.png)

A pipeline describes how things need to be executed in order to achieve wanted result. In CDS, a pipeline a defined on a project and can be used on several applications inside the same project.

A pipeline is structured in sequential **[stages]({{< relref "introduction.concepts.stage.md" >}})** containing one or multiple concurrent **[jobs]({{< relref "introduction.concepts.job.md" >}})**.

In CDS there is several types of pipeline : **build**, **testing** and **deployment**. In Pipeline configuration file, default type is **build**.

The goal is to make your pipeline the more reusable as possible. It have to be able to build, test or deploy all the tiers, services or micro-services of your project.

You can also define ACL on a pipeline.

## Triggers

![Triggers](/images/concepts_pipeline_trigger.png)

## Example

![Example](/images/concepts_pipeline_example.png)
