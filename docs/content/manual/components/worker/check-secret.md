+++
title = "check-secret"
+++
## worker check-secret

`worker check-secret fileA fileB`

### Synopsis



Inside a step script (https://ovh.github.io/cds/workflows/pipelines/actions/builtin/script/), you can add check if a file contains a CDS variable of type password or private key:

```bash
#!/bin/bash

set -ex

# create a file
cat << EOF > myFile
this a a line in the file, with a CDS variable of type password {{.cds.app.password}}
EOF

# worker check-secret myFile
worker check-secret {{.cds.workspace}}/myFile
```

This command will exit 1 and a log is displayed, as:

	variable cds.app.password is used in file myFile

The command will exit 0 if no variable of type password or key is found.

		

```
worker check-secret [flags]
```

### Options

```
  -h, --help   help for check-secret
```

### SEE ALSO

* [worker](/cli/worker/worker/)	 - CDS Worker

