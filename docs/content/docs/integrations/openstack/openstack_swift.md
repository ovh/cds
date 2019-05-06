---
title: OpenStack Swift
main_menu: true
card: 
  name: storage
---

The OpenStack Swift Integration is a Self-Service integration that can be configured on a CDS Project.

With this integration, you can use a dedicated OpenStack Swift Storage on :

- action [Artifact Upload]({{< relref "/docs/actions/artifact-upload.md">}})
- action [Artifact Download]({{< relref "/docs/actions/artifact-download.md">}})
- action [Serve Static Files]({{< relref "/docs/actions/serve-static-files.md">}})
- [worker cache command]({{< relref "/docs/components/worker/cache">}})

Notice: by default, the storage is configured in CDS Configuration. This integration
allows user to use their own OpenStack Storage and not use the shared storage.

## Configure with WebUI

You can add a OpenStack Swift on your CDS Project.

![Integration](../images/openstack-swift-integration-webui.png)

## Configure with cdsctl

### Import a OpenStack Swift on your CDS Project

Create a file project-configuration.yml:

```yml
name: MyOpenstackTenant
model:
  name: Openstack
  public: false
config:
  address:
    value: https://auth.cloud.ovh.net/v2.0/
    type: string
  domain:
    value: ""
    type: string
  password:
    value: 'your-password-here'
    type: password
  region:
    value: your-region
    type: string
  storage_container_prefix:
    value: cds-prefix-
    type: string
  storage_temporary_url_supported:
    value: "true"
    type: string
  tenant_name:
    value: "your-openstack-tenant"
    type: string
  username:
    value: your-openstack-user
    type: string
```

Import the integration on your CDS Project with:

```bash
cdsctl project integration import PROJECT_KEY project-configuration.yml
```

### Create a Public OpenStack Swift for whole CDS Projects

You can also add a OpenStack Swift with cdsctl. As a CDS Administrator,
this allows you to propose a Public OpenStack Swift, available on all CDS Projects.

Create a file public-configuration.yml:

```yml
name: Openstack
storage: true
public: true
public_configurations:
  your-public-openstack-integration:
    "address":
      value: https://auth.cloud.ovh.net/v2.0/
      type: string
    "domain":
      value: ""
      type: string
    "password":
      value: 'your-password-here'
      type: password
    "region":
      value: your-region
      type: string
    "storage_container_prefix":
      value: cds-prefix-
      type: string
    "storage_temporary_url_supported":
      value: "true"
      type: string
    "tenant_name":
      value: "your-openstack-tenant"
      type: string
    "username":
      value: your-openstack-user
      type: string
```

Import the integration with :

```bash
cdsctl admin integration-model import public-configuration.yml
```
