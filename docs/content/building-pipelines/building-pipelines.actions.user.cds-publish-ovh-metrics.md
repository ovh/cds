+++
title = "cds-publish-ovh-metrics"

[menu.main]
parent = "actions-user"
identifier = "cds-publish-ovh-metrics"

+++

Publish a metric on OVH Metrics. See https://www.ovh.com/fr/data-platforms/metrics/ and doc on https://docs.ovh.com/gb/en/cloud/metrics/

## Parameters

* **file**: Metrics file to push (optional) See: http://www.warp10.io/apis/ingress/
* **labels**: Labels of your metric (one K/V per line separated by a space)
* **name**: Name of you metric (optional)
* **region**: Metrics region
* **token**: Metrics write token
* **value**: Value of your metric (T=true) See: http://www.warp10.io/apis/ingress/


## Requirements

* **bash**: type: binary Value: bash
* **curl**: type: binary Value: curl


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-publish-ovh-metrics.hcl)


