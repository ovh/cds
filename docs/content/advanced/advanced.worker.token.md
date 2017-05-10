+++
title = "Generate a Token"
weight = 1

[menu.main]
parent = "advanced"
identifier = "advanced-worker-token"

+++

### Purpose

In order to start a worker or a Hatchery, you need to provide a token to be able to register on CDS API.

### CLI

Run the following command, replace yourgroup with your group

```bash
$ cds generate token -g yourgroup -e persistent
```
