+++
title = "User Actions"
weight = 1

+++

A user action is a combination of built-in or plugin actions. A user can import user actions using the Web UI or the CLI:

## Import an action using the CLI

With a local file  :

```bash
$ cds action add --url $GOPATH/src/github.com/ovh/cds/contrib/actions/actions/cds-docker-package.hcl
```

With a remote file  :

```bash
$ cds action add --url https://raw.githubusercontent.com/ovh/cds/master/contrib/actions/cds-docker-package.hcl
```


CDS's source code bundles a few user-actions that you may use directly or as a starting point for your own user-actions:

{{%children style=""%}}
