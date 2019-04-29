---
title: AWS S3
main_menu: true
card: 
  name: storage
---

The AWS S3 Integration is a Self-Service integration that can be configured on a CDS Project.

With this integration, you can use a dedicated AWS S3 Bucket Storage on :

- action [Artifact Upload]({{< relref "/docs/actions/artifact-upload.md">}})
- action [Artifact Download]({{< relref "/docs/actions/artifact-download.md">}})
- [worker cache command]({{< relref "/docs/components/worker/cache">}})

Notice: by default, the storage is configured in CDS Configuration. This integration
allows user to use their own AWS S3 Storage and not use the shared storage.

## Configure with WebUI

You can add a AWS S3 on your CDS Project.

![Integration](../images/aws-s3-integration-webui.png)

## Configure with cdsctl

### Import a AWS S3 on your CDS Project

Create a file project-configuration.yml:

```yml
name: MyAWS
model:
  name: AWS
  public: false
config:
  region:
    value: your-region
    type: string
  bucket_name:
    value: your-bucket-name
    type: string
  prefix:
    value: cds-prefix-
    type: string
  access_key_id:
    value: your-access-key
    type: string
  secret_access_key:
    value: 'your-secret-access-key'
    type: password
```

Import the integration on your CDS Project with:

```bash
cdsctl project integration import PROJECT_KEY project-configuration.yml
```

### Create a Public AWS S3 for whole CDS Projects

You can also add a AWS S3 with cdsctl. As a CDS Administrator,
this allows you to propose a Public AWS S3, available on all CDS Projects.

Create a file public-configuration.yml:

```yml
name: AWS
storage: true
public: true
public_configurations:
  your-public-aws-s3-integration:
    "region":
      value: your-region
      type: string
    "bucket_name":
      value: your-bucket-name
      type: string
    "prefix":
      value: cds-prefix-
      type: string
    "access_key_id":
      value: your-access-key
      type: string
    "secret_access_key":
      value: 'your-secret-access-key'
      type: password
```

Import the integration with :

```bash
cdsctl admin integration-model import public-configuration.yml
```
