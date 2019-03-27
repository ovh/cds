---
title: "tmpl"
notitle: true
notoc: true
---
# worker tmpl

`worker tmpl inputFile outputFile`

## Synopsis



Inside a step script (https://ovh.github.io/cds/docs/actions/script/), you can add a replace CDS variables with the real value into a file:

	# create a file
	cat << EOF > myFile
	this a a line in the file, with a CDS variable {{.cds.version}}
	EOF

	# worker tmpl <input file> <output file>
	worker tmpl {{.cds.workspace}}/myFile {{.cds.workspace}}/outputFile


The file `outputFile` will contain the string:

	this a a line in the file, with a CDS variable 2


if it's the RUN n°2 of the current workflow.
		

```
worker tmpl
```

## SEE ALSO

* [worker](/docs/components/worker/worker/)	 - CDS Worker

