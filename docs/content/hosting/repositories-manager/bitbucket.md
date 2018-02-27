+++
title = "Bitbucket"
weight = 2

+++

## Authorize CDS on your Bitbucket instance
You need to perform the following steps :

 - Bitbucket admin privileges
 - A RSA Key Pair

### Create a CDS application in BitBucket
In Bitbucket go to *Administration Settings* / *Application Links*. Create a new Application with :

 - Name : **CDS**
 - Type : **Generic Application**
 - Application URL : *Your CDS URL*
 - Display URL : *Your CDS URL*

On this application, you just have to set up *OAuth Incoming Authentication* :

 - Consumer Key : **CDS** (you can change it in your configuration file)
 - Consumer Name : **CDS**
 - Public Key : *Your CDS RSA public key*
 - Consumer Callback URL : None
 - Allow 2-Legged OAuth : false
 - Execute as : None
 - Allow user impersonation through 2-Legged OAuth : false

### Complete CDS Configuration File

Set value to `privateKey`. You can modify `consumerKey` if you want.

```yaml
 [vcs.servers]

    [vcs.servers.Bitbucket]

      # URL of this VCS Server
      url = "https://mybitbucket.localhost"

      [vcs.servers.Bitbucket.bitbucket]
        # you can change the consumeKey if you want
        consumerKey = "CDS"

        # Does polling is supported by VCS Server
        disablePolling = false

        # Does webhooks are supported by VCS Server
        disableWebHooks = false

        privateKey = "-----BEGIN PRIVATE KEY-----\n....\n-----END PRIVATE KEY-----"

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

        [vcs.servers.mybitbucket_instance2.bitbucket.Status]

          # Set to true if you don't want CDS to push statuses on the VCS server
          disable = false

          # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server
          showDetail = true

```

**Then restart CDS**

See how to generate **[Configuration File]({{<relref "/hosting/configuration/_index.md" >}})**