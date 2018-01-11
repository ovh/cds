+++
title = "Worker Tmpl"
weight = 4

+++

Inside a step [script]({{< relref "workflows/pipelines/actions/builtin/script.md" >}}), you can add a replace CDS variables with the real value into a file:

```bash

# create a file
cat << EOF > myFile
this a a line in the file, with a CDS variable {{.cds.version}}
EOF

# worker tmpl <input file> <output file>
worker tmpl {{.cds.workspace}}/myFile {{.cds.workspace}}/outputFile
```

The file `outputFile` will contain the string:

```
this a a line in the file, with a CDS variable 2
```

if it's the RUN n°2 of the current workflow.