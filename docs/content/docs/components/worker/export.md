---
title: "export"
notitle: true
notoc: true
---
# worker export

`worker export <varname> <value>`

## Synopsis


Inside a step script (https://ovh.github.io/cds/docs/actions/script/), you can create a build variable with the worker command:

	worker export foo bar


then, you can use new build variable:

	echo "{{.cds.build.foo}}"


## Scope

You can use the build variable in :

* another step of the current job with `{{.cds.build.varname}}`
* the next stages in same pipeline `{{.cds.build.varname}}`
* the next pipelines `{{.workflow.pipelineName.build.varname}}` with `pipelineName` the name of the pipeline in your workflow
	
	

```
worker export
```

## SEE ALSO

* [worker](/docs/components/worker/worker/)	 - CDS Worker

