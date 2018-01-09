+++
title = "Worker Export"
weight = 2

[menu.main]
parent = "commands"
identifier = "worker.export"

+++

Inside a step [script]({{< relref "workflows/pipelines/actions/builtin/script.md" >}}), you can create a build variable with the worker command:

```bash
# worker export <varname> <value>
worker export foo bar
```

then, you can use new build variable:

```bash
echo "{{.cds.build.foo}}"
```

## Scope

You can use the build variable in :

 * the current job with `{{.cds.build.varname}}`
 * the next stages in same pipeline `{{.cds.build.varname}}`
 * the next pipelines `{{.workflow.pipelineName.build.varname}}` with `pipelineName` the name of the pipeline in your worklow
