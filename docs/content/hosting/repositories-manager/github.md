+++
title = "Github"
weight = 1

+++

## Authorize CDS on Github
### Create a CDS application on Github
Go to `https://github.com/settings/developers` and **Register a new application**: set an application name, the url and a description. `Authorization callback URL`: `http(s)://<your-cds-api>/repositories_manager/oauth2/callback`

On the next page Github give you a **Client ID** and a **Client Secret**

### Complete CDS Configuration File

Set value to `clientId` and `clientSecret`

```yaml
    [vcs.servers.github]

      # URL of this VCS Server
      url = "https://github.com"

      [vcs.servers.github.github]

        # github OAuth Application Client ID
        clientId = "xxxx"

        # github OAuth Application Client Secret
        clientSecret = "xxxx"

        # Does polling is supported by VCS Server
        disablePolling = false

        # Does webhooks are supported by VCS Server
        disableWebHooks = false

        [vcs.servers.github.github.Status]

          # Set to true if you don't want CDS to push statuses on the VCS server
          disable = false

          # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server
          showDetail = true
```

**Then restart CDS**

See how to generate **[Configuration File]({{<relref "/hosting/configuration/_index.md" >}})**