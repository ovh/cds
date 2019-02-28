+++
title = "cache"
+++
## worker cache



### Synopsis


Inside a project, you can create or retrieve a cache from your worker with a tag (useful for vendors for example).

You can access to this cache from any workflow inside a project. You just have to choose a tag that fits with your needs.

For example if you need a different cache for each workflow so choose a tag scoped with your workflow name and workflow version (example of tag value: {{.cds.workflow}}-{{.cds.version}})

## Use Case
Java Developers often use maven to manage dependencies. The mvn install command could be long because all the maven dependencies have to be downloaded on a fresh CDS Job workspace.
With the worker cache feature, you don't have to download the dependencies if they haven't been updated since the last run of the job.


- cache push: take the current .m2/ directory and set it as a cache
- cache pull: download a cache of .m2 directory

Here, an example of a script inside a CDS Job using the cache feature:

	#!/bin/bash

	tag=($(md5sum pom.xml))

	# download the cache of .m2/
	if worker cache pull $tag; then
		echo ".m2/ getted from cache";
	fi

	# update the directory .m2/
	# as there is a cache, mvn does not need to download all dependencies
	# if they are not updated on upstream
	mvn install

	# put in cache the updated .m2/ directory
	worker cache push $tag .m2/

    

### Options

```
  -h, --help   help for cache
```

### SEE ALSO

* [worker](/cli/worker/worker/)	 - CDS Worker
* [worker cache pull](/cli/worker/cache/pull/)	 - `worker cache pull tagValue`
* [worker cache push](/cli/worker/cache/push/)	 - `worker cache push tagValue {{.cds.workspace}}/pathToUpload`

