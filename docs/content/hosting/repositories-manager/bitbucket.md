+++
title = "Bitbucket"
weight = 2

[menu.main]
parent = "repositories-manager"
identifier = "repositories-manager.bitbucket"

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

### Connect CDS To Bitbucket
With CDS CLI run :

 ```
 $ cds admin reposmanager add STASH mystash.mynetwork.net http://mystash.mynetwork.net key=privatekey
 ```

And follow instructions.

Set in your configuration file the CDS **private key**

Restart CDS.

Now check everything is OK with :
 ```
 $ cds admin reposmanager list
 ```
