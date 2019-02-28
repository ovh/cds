+++
title = "pull"
+++
## worker cache pull

`worker cache pull tagValue`

### Synopsis


Inside a project, you can fetch a cache from your worker with a tag

	worker pull <tagValue>

If you push a cache with:

	worker cache push latest {{.cds.workspace}}/pathToUpload

The command:

	worker cache pull latest

will create the directory {{.cds.workspace}}/pathToUpload with the content of the cache

		

```
worker cache pull [flags]
```

### Options

```
  -h, --help   help for pull
```

### SEE ALSO

* [worker cache](/cli/worker/cache/)	 - 

