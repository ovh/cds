---
title: "Coverage"
card:
  name: builtin
---

**Coverage** is a builtin action, you can't modify it.

CDS Builtin Action.
Parse given file to extract coverage results.

Coverage report will be linked to the application from the pipeline context.
You will be able to see the coverage history in the application home page.

## Parameters

* **format**: Coverage report format.
* **minimum**: Minimum percentage of coverage required (-1 means no minimum).
* **path**: Path of the coverage report file.


## Requirements

No Requirement

## YAML example

Example of a pipeline using Coverage action:
```yml
version: v1.0
name: Pipeline1
stages:
- Stage1
jobs:
- job: Job1
  stage: Stage1
  steps:
  - coverage:
      format: cobertura
      minimum: ./coverage.xml
      path: "60"

```

