+++
title = "push"
+++
## cdsctl workflow push

`Push a workflow`

### Synopsis


Useful when you want to push a workflow and his dependencies (pipelines, applications, environments)

For example if you have a workflow with pipelines build and tests you can push your workflow and pipelines with

	cdsctl workflow push tests.pip.yml build.pip.yml myWorkflow.yml

	

```
cdsctl workflow push [ PROJECT-KEY ] YAML-FILE ... [flags]
```

### Options

```
  -h, --help                help for push
      --skip-update-files   Useful if you don't want to update yaml files after pushing the workflow.
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl workflow](/manual/components/cdsctl/workflow/)	 - `Manage CDS workflow`

