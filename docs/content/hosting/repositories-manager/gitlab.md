+++
title = "Gitlab"
weight = 2

+++

## Authorize CDS on your Gitlab instance
What you need to perform the following steps :

 - Gitlab admin privileges

### Create a CDS application on Gitlab
In Gitlab go to *Settings* / *Application* section. Create a new application with :

 - Name : **CDS**
 - Redirect URI : **https://your-cds-api/repositories_manager/oauth2/callback**

Scopes :

 - API
 - read_user
 - read_registry

### Complete CDS Configuration File

Set value to `appId` and `secret`


```yaml
   [vcs.servers.Gitlab]

      # URL of this VCS Server
      url = "https://gitlab.com"

      [vcs.servers.Gitlab.gitlab]
        appId = "xxxx"

        # Does polling is supported by VCS Server
        disablePolling = false

        # Does webhooks are supported by VCS Server
        disableWebHooks = false

        secret = "xxxx"

        # If you want to have a reverse proxy url for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK
        # proxyWebhook = "https://myproxy.com"

        [vcs.servers.Gitlab.gitlab.Status]

          # Set to true if you don't want CDS to push statuses on the VCS server
          disable = false

          # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server
          showDetail = true
```

**Then restart CDS**

See how to generate **[Configuration File]({{<relref "/hosting/configuration/_index.md" >}})**
