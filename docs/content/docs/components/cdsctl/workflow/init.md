---
title: "init"
notitle: true
notoc: true
---
# cdsctl workflow init

`Init a workflow`

## Synopsis

Initialize a workflow from your current repository, this will read or create yml files and push them to CDS.

Documentation: https://ovh.github.io/cds/docs/tutorials/init_workflow_with_cdsctl/



```
cdsctl workflow init [PROJECT-KEY] [flags]
```

## Options

```
      --application string           (Optional) Set the application name. If empty, it will deduce application name from the repository.
      --pipeline string              (Optional) Set the root pipeline you want to use. If empty it will propose you to reuse of create a pipeline.
      --repository-fullname string   (Optional) Set the repository fullname defined in repository manager
      --repository-pgp-key string    Set the repository pgp key you want to use
      --repository-ssh-key string    Set the repository access key you want to use
      --repository-url string        (Optional) Set the repository remote URL. Default is the fetch URL
      --workflow string              (Optional) Set the workflow name. If empty, it will deduce workflow name from the repository.
  -y, --yes                          Automatic yes to prompts. Assume "yes" as answer to all prompts and run non-interactively.
```

## Options inherited from parent commands

```
  -f, --file string   set configuration file
      --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
      --verbose       verbose output
```

## SEE ALSO

* [cdsctl workflow](/docs/components/cdsctl/workflow/)	 - `Manage CDS workflow`

