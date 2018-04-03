+++
title = "User Actions"
weight = 1

+++

A user action is a combination of built-in or plugin actions. A user can import user actions using the Web UI or the CLI:

## Import an action using the CLI

```bash
$ git clone https://github.com/ovh/cds.git
$ cd cds/contrib/actions/
$ cdsctl action import cds-docker-package.yml
```

See [cdsctl action import]({{< relref "/cli/cdsctl/action/import.md" >}}) documentation.

CDS's source code bundles a few user-actions that you may use directly or as a starting point for your own user-actions:

{{%children style=""%}}
