---
title: "import"
notitle: true
notoc: true
---
# cdsctl admin metadata application import

`import CDS Application Metadata`

## Synopsis

Metadata are represented with key:value

Example of a csv file for a CDS Application
	
	project_key;application_name;last_modified;vcs_repofullname;ou1;ou2
	YOUR_PROJECT_KEY;Your Application Name;2020-01-01T00:00:00;repo_of_application;OU_1_VALUE;OU_2_VALUE

You can enter as many metadata as desired, the key name is on the first line of the csv file.


```
cdsctl admin metadata application import PATH
```

## Options inherited from parent commands

```
  -f, --file string   set configuration file
      --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
      --verbose       verbose output
```

## SEE ALSO

* [cdsctl admin metadata application](/docs/components/cdsctl/admin/metadata/application/)	 - `Manage CDS Application Metadata`

