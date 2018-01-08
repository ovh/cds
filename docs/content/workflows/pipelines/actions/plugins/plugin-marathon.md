+++
title = "plugin-marathon"

[menu.main]
parent = "plugins"
identifier = "plugin-marathon"

+++

This action helps you to deploy on Mesos/Marathon. Provide a marathon.json file to configure deployment.

Your marathon.json file can be templated with cds variables "{{.cds.variables}}". Enable "waitForDeployment" option to ensure deployment is successfull.

## Parameters

* **configuration**: Marathon application configuration file (json format)
* **insecureSkipVerify**: Skip SSL Verify if you want to use self-signed certificate
* **password**: Marathon Password (please use project, application or environment variables)
* **timeout**: Marathon deployment timeout (seconds). Used only if "waitForDeployment" is true. 
* **url**: Marathon URL http://127.0.0.1:8081,http://127.0.0.1:8082,http://127.0.0.1:8083
* **user**: Marathon User (please use project, application or environment variables)
* **waitForDeployment**: Wait for instances deployment.
If set, CDS will wait for all instances to be deployed until timeout is over. All instances deployment must be done to get a successful result.
If not set, CDS will consider a successful result if marathon accepts the provided configuration.


## More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/plugins/plugin-marathon/README.md)

