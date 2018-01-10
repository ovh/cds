+++
title = "Stage"
weight = 2

[menu.main]
parent = "concepts"
identifier = "concepts.stage"

+++


Usually in CDS a **build** pipeline is structured of the following stages :

- **Compile stage** : Build the binaries
- **Analysis & Unit Tests stage** : Run all unit tests and analyse code quality
- **Packaging stage** : Build the final package, Virtual Machine Image or Docker Image.

In CDS, stages are executed sequentially if the previous stage is successful.

You can define trigger conditions on a stage, to enable/disable it under given conditions. For instance, you may want to run the *Compile Stage* and *Analysis & Unit Tests stage* on all branches but dedicate the *Packaging Stage* run on `master` and `develop` branches only.

A **Stage** is a set of jobs which will be run in parallel.

![Pipeline](/images/concepts_pipeline.png)
