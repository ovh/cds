---
title: "import"
notitle: true
notoc: true
---
# cdsctl admin metadata project import

`import CDS Project Metadata`

## Synopsis

Metadata are represented with key:value

Example of a csv file for a CDS Project
	
	project_key;project_name;last_modified;ou1;ou2
	YOUR_PROJECT_KEY;Your Project Name;2020-01-01T00:00:00;OU_1_VALUE;OU_2_VALUE

You can enter as many metadata as desired, the key name is on the first line of the csv file.


```
cdsctl admin metadata project import PATH
```

## Options inherited from parent commands

```
  -f, --file string   set configuration file
      --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
      --verbose       verbose output
```

## SEE ALSO

* [cdsctl admin metadata project](/docs/components/cdsctl/admin/metadata/project/)	 - `Manage CDS Project Metadata`

