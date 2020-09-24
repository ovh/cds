---
title: Bitbucket Server
main_menu: true
card: 
  name: repository-manager
---

The Bitbucket Server Integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to link a Git Repository hosted by your Bitbucket Server
to a CDS Application.

This integration enables some features:

 - [Git Repository Webhook]({{<relref "/docs/concepts/workflow/hooks/git-repo-webhook.md" >}})
 - Easy to use action [CheckoutApplication]({{<relref "/docs/actions/builtin-checkoutapplication.md" >}}) and [GitClone]({{<relref "/docs/actions/builtin-gitclone.md">}}) for advanced usage
 - Send [build notifications](https://developer.atlassian.com/server/bitbucket/how-tos/updating-build-status-for-commits/) on your Pull-Requests and Commits on Bitbucket. [More informations]({{<relref "/docs/concepts/workflow/notifications.md#vcs-notifications" >}})
 - [Send comments on your Pull-Requests when a workflow is failed]({{<relref "/docs/concepts/workflow/notifications.md#vcs-notifications" >}})

## How to configure Bitbucket Server integration

You need to perform the following steps:

 - Bitbucket admin privileges
 - A RSA Key Pair

### Generate RSA Key Pair

Create the private RSA certificate:

```
$ openssl genrsa -out key.pem 1024
```

The content of key.pem have to be used as `privateKey` below in CDS Configuration file.

Generate the public RSA certificate:

```
$ openssl rsa -in key.pem -pubout
```

This will display the public key, you will have to copy-paste it inside `Public Key` field on Bitbucket.


### Create a CDS application in Bitbucket
In Bitbucket go to *Administration Settings* / *Application Links*. Create a new Application with:

 - Name: **CDS**
 - Type: **Generic Application**
 - Application URL: *Your CDS API URL*
 - Display URL: *Your CDS API URL*

On this application, you just have to set up *OAuth Incoming Authentication*:

 - Consumer Key: **CDS** (you can change it in your configuration file)
 - Consumer Name: **CDS**
 - Public Key: *Your CDS RSA public key*
 - Consumer Callback URL: None
 - Allow 2-Legged OAuth: false
 - Execute as: None
 - Allow user impersonation through 2-Legged OAuth: false

### Complete CDS Configuration File

Set value to `privateKey`. You can modify `consumerKey` if you want.

```yaml
 [vcs.servers]

    [vcs.servers.Bitbucket]

      # URL of this VCS Server
      url = "https://mybitbucket.com"

      [vcs.servers.Bitbucket.bitbucket]

        #######
        # CDS <-> Bitbucket. Documentation on https://ovh.github.io/cds/hosting/repositories-manager/bitbucket/
        ########
        # You can change the consumeKey if you want
        consumerKey = "CDS"

        # Does polling is supported by VCS Server
        disablePolling = false

        # Does webhooks are supported by VCS Server
        disableWebHooks = false
        privateKey = "-----BEGIN PRIVATE KEY-----\n....\n-----END PRIVATE KEY-----"

        # If you want to have a reverse proxy URL for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK
        # proxyWebhook = ""

        # optional, Bitbucket Token associated to username, used to add comment on Pull Request
        token = ""

        # optional. Bitbucket username, used to add comment on Pull Request on failed build.
        username = ""

        [vcs.servers.Bitbucket.bitbucket.Status]

          # Set to true if you don't want CDS to push statuses on the VCS server
          # disable = false
```

You can configure many instances of Bitbucket:


```yaml

[vcs.servers]

    [vcs.servers.mybitbucket_instance1]

      # URL of this VCS Server
      url = "https://mybitbucket-instance1.localhost"

      [vcs.servers.mybitbucket_instance1.bitbucket]
        consumerKey = "CDS_Instance1"

        # Does polling is supported by VCS Server
        disablePolling = true

        # Does webhooks are supported by VCS Server
        disableWebHooks = false

        # Does webhooks creation are supported by VCS Server
        disableWebHooksCreation = false
        privateKey = "-----BEGIN PRIVATE KEY-----\n....\n-----END PRIVATE KEY-----"

        # If you want to have a reverse proxy URL for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK
        # proxyWebhook = "https://myproxy.com"

        [vcs.servers.mybitbucket_instance1.bitbucket.Status]

          # Set to true if you don't want CDS to push statuses on the VCS server
          disable = false

          # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server
          showDetail = true

    [vcs.servers.mybitbucket_instance2]

      # URL of this VCS Server
      url = "https://mybitbucket-instance2.localhost"

      [vcs.servers.mybitbucket_instance2.bitbucket]
        consumerKey = "CDS_Instance2"

        # Does polling is supported by VCS Server
        disablePolling = true

        # Does webhooks are supported by VCS Server
        disableWebHooks = false

        # Does webhooks creation are supported by VCS Server
        disableWebHooksCreation = false
        privateKey = "-----BEGIN PRIVATE KEY-----\n....\n-----END PRIVATE KEY-----"

        # If you want to have a reverse proxy URL for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK
        # proxyWebhook = "https://myproxy.com"

        [vcs.servers.mybitbucket_instance2.bitbucket.Status]

          # Set to true if you don't want CDS to push statuses on the VCS server
          disable = false

          # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server
          showDetail = true

```

See how to generate **[Configuration File]({{<relref "/hosting/configuration.md" >}})**

## Start the vcs µService

```bash
$ engine start vcs

# you can also start CDS api and vcs in the same process:
$ engine start api vcs
```

## Vcs events

For now, CDS supports push events. CDS uses this push event to remove existing runs for deleted branches (24h after branch deletion).
