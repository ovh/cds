---
title: "Region Requirement"
weight: 4
---

The `Region` prerequisite allows you to require a worker to have access to a specific region.

A `Region` can be configured on each hatchery. With a free text as `myregion` in hatchery configuration, 
user can set a prerequisite 'region' with value `myregion` on CDS Job.

Example of job configuration:
```
jobs:
- job: build
  requirements:
  - region: myregion
  steps:
  ...
```
