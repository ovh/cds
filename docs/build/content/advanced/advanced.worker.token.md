+++
title = "Token"
weight = 3

[menu.main]
parent = "advanced-worker"
identifier = "advanced-worker-token"

+++


Generate a Token
=====================

### Purpose

In order to start a worker, you need to provide a worker key to be able to build your pipelines.

### CLI

Run the following command, replace yourgroup with your group
```bash
$ cds generate token -g yourgroup -e persistent
```
