
name = "CDS_PublishOVHMetrics"
description = "Publish a metric on OVH Metrics. See https://www.ovh.com/fr/data-platforms/metrics/ and doc on https://docs.ovh.com/gb/en/cloud/metrics/"

// Requirements
requirements = {
  "curl" = {
		type = "binary"
		value = "curl"
	}
	"bash" = {
		type = "binary"
		value = "bash"
	}
}

// Parameters
parameters = {
	 "name" = {
		type = "string"
		description = "Name of you metric (optional)"
		value = "cds"
	}
	 "labels" = {
		type = "text"
		description = "Labels of your metric (one K/V per line separated by a space)"
		value = "app {{.cds.application}}
env {{.cds.environment}}"
	}
	 "value" = {
		type = "string"
		description = "Value of your metric (T=true) See: http://www.warp10.io/apis/ingress/"
		value = "T"
	}
	 "file" = {
		type = "string"
		description = "Metrics file to push (optional) See: http://www.warp10.io/apis/ingress/"
	}
	 "region" = {
		type = "string"
		description = "Metrics region"
		value = "gra1"
	}
	 "token" = {
		type = "string"
		description = "Metrics write token"
	}
}

// Steps
steps = [{
	script = <<EOF
#!/bin/bash

set -e

if [ "{{.name}}" != "" ]; then

labels=`cat << EOF | sed 's/ /%20/g' | sed 's/%20/=/1' | tr '\n' ',' | sed 's/, *$//'
{{.labels}}
EOF`

echo "// {{.name}}{$labels} {{.value}}" >> .metrics

curl -f -X POST https://warp10.{{.region}}.metrics.ovh.net/api/v0/update \
		-H 'X-Warp10-Token: {{.token}}' \
    --data-binary @.metrics

fi;


if [ "{{.file}}" != "" ]; then
curl -f -X POST https://warp10.{{.region}}.metrics.ovh.net/api/v0/update \
		-H 'X-Warp10-Token: {{.token}}' \
    --data-binary @{{.file}}
fi;
EOF
	}]
