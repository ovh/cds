+++
title = "GitLab"
weight = 2

+++

## Authorize CDS on your GitLab instance
What you need to perform the following steps:

 - GitLab admin privileges

### Create a CDS application on GitLab
In GitLab go to *Settings* / *Application* section. Create a new application with:

 - Name: **CDS**
 - Redirect URI: **https://your-cds-api/repositories_manager/oauth2/callback**

Scopes:

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

        #######
        # CDS <-> GitLab. Documentation on https://ovh.github.io/cds/hosting/repositories-manager/gitlab/
        ########
        appId = "xxxx"

        # Does polling is supported by VCS Server
        disablePolling = false

        # Does webhooks are supported by VCS Server
        disableWebHooks = false

        # If you want to have a reverse proxy URL for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK
        # proxyWebhook = ""
        secret = "xxxx"

        [vcs.servers.Gitlab.gitlab.Status]

          # Set to true if you don't want CDS to push statuses on the VCS server
          # disable = false

          # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server
          # showDetail = false
```

**Then restart CDS**

See how to generate **[Configuration File]({{<relref "/hosting/configuration/_index.md" >}})**
