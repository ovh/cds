+++
title = "push"
+++
## worker cache push

`worker cache push tagValue {{.cds.workspace}}/pathToUpload`

### Synopsis


Inside a project, you can create a cache from your worker with a tag (useful for vendors for example)
	worker push <tagValue> dir/file
		

```
worker cache push [flags]
```

### Examples

```
worker cache push {{.cds.workflow}}-{{.cds.version}} {{.cds.workspace}}/pathToUpload
```

### Options

```
  -h, --help   help for push
```

### SEE ALSO

* [worker cache](/cli/worker/cache/)	 - 

