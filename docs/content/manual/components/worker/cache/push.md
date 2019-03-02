+++
title = "push"
+++
## worker cache push

`worker cache push tagValue {{.cds.workspace}}/pathToUpload`

### Synopsis


Inside a project, you can create a cache from your worker with a tag (useful for vendors for example)
	worker push <tagValue> dir/file

You can use you storage integration: 
	worker push --destination=MyStorageIntegration  <tagValue> dir/file
		

```
worker cache push [flags]
```

### Examples

```
worker cache push {{.cds.workflow}}-{{.cds.version}} {{.cds.workspace}}/pathToUpload
```

### Options

```
      --destination string   optional. Your storage integration name
  -h, --help                 help for push
```

### SEE ALSO

* [worker cache](/manual/components/worker/cache/)	 - 

