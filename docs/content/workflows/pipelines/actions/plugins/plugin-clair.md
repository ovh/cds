+++
title = "plugin-clair"

+++

This plugin analyze your docker image using clair (https://github.com/coreos/clair)

Add an extra step of type plugin-clair on your job to use it.

## Parameters

* **image**: Image to analize

### Prerequisites

To use this plugin, you must :

* Have clair running: https://github.com/coreos/clair/blob/master/Documentation/running-clair.md
* Add clair in your CDS configuration: 

```yml
[[api.services]]
        healthUrl = "http://localhost"
        healthPort = "6061"
        healthPath = "/health"
        name = "clair"
        type = "clair" # MUST BE 'clair'
        url = "http://localhost"
        port = "6060"
        path = ""
```

