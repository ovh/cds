---
title: AWS S3
main_menu: true
card: 
  name: storage
---

The AWS S3 Integration is a Self-Service integration that can be configured on a CDS Project.

With this integration, you can use a dedicated AWS S3 Bucket Storage on :

- action [Artifact Upload]({{< relref "/docs/actions/builtin-artifact-upload.md">}})
- action [Artifact Download]({{< relref "/docs/actions/builtin-artifact-download.md">}})
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

### Using min.io as an alternative

[Minio](https://min.io) is a Open Source, Enterprise-Grade, Amazon S3 Compatible Object Storage.

According to https://docs.min.io/docs/how-to-use-aws-sdk-for-go-with-minio-server.html, you can define `endpoint`, `disable_ssl`, `force_path_style` to link CDS to a Minio server.

For example, you can run a minio local server with the following docker command.

```bash
docker run -p 9000:9000 --name minio1 \
  -e "MINIO_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE" \
  -e "MINIO_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" \
  -v /mnt/data:/data \
  minio/minio server /data
```

Then you can import the following content to you project with the `cdsctl project integration import` command.

```yaml
name: local min.io
model:
  name: AWS
storage: true
config:
  region:
    value: us-east-1
    type: string
  bucket_name:
    value: cds-storage
    type: string
  prefix:
    value: cds-prefix-
    type: string
  access_key_id:
    value: 'AKIAIOSFODNN7EXAMPLE'
    type: string
  secret_access_key:
    value: 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY'
    type: password
  endpoint:
    value: 'http://localhost:9000'
    type: string
  disable_ssl:
    value: 'true'
    type: boolean
  force_path_style:
    value: 'true'
    type: boolean
```