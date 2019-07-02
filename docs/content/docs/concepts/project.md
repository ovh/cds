---
title: "Project"
weight: 2
card: 
  name: concept_organization
---


A CDS Project brings together several entities such as pipelines, applications, workflows, environments. 
A project also allows to link one or more repository manager such as github, bitbucket, gitlab...

A project is often associated with one or more groups with differents ACLs.

## Metadata

In a company, a project is a collection of a team's workflows. This allows to release some statistics of use with metadata.

A metadata is composed by a key and a value.

Export CDS Projects:

```bash
cdsctl admin metadata project export
## a file export_metadata_projects.csv is created
```

The file `export_metadata_projects.csv` looks like :

```csv
project_key;project_name;last_modified;ou1;ou2
PRJ_KEY_A;My Fist Project;2019-06-21T17:52:36;foo;bar
PRJ_KEY_B;Project B;2019-06-21T17:52:36;foo;bar2
PRJ_KEY_C;Project C;2019-06-21T17:52:37;foo;bar2
```

Here, `ou1` and `ou2` are two metadata key. You can add more metadata by adding key on the first line and values on lines below.

You can import / create metadata:

```
cdsctl admin metadata project import export_metadata_projects.csv
```

Notice that exporting metadata on appliation & workflows will export metadata from project. On the example above, the metadata `ou1` is setted on all workflows and applications on the third projects.
