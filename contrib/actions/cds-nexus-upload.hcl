
name = "CDS_NexusUpload"
description = "Upload file on Nexus"

// Requirements
requirements = {
	"bash" = {
		type = "binary"
		value = "bash"
	}
	"login" = {
		type = "string"
	}
	"password" = {
		type = "string"
	}
	"url" = {
		type = "string"
		description = "Nexus URL"
	}
	"repository" = {
		type = "string"
	}
	"extension" = {
		type = "string"
	}
	"groupId" = {
		type = "string"
		value = "{{.cds.application}}"
	}
	"artifactId" = {
		type = "string"
		value = "{{.cds.application}}"
	}
	"version" = {
		type = "string"
		value = "{{.cds.build.VERSION}}"
	}
	"file" = {
		type = "string"
	}
	"packaging" = {
		type = "string"
	}
}

// Parameters
parameters = {
	
}

// Steps
steps = [{
	script = <<EOF
#!/bin/bash
set -e

echo "action git from directory"
pwd
echo "Upload to Nexus ({{.url}}) on repository {{.repository}}"
curl -F r={{.repository}} -F hasPom=false -F e={{.extension}} -F g="{{.groupId}}" -F a="{{.artifactId}}" -F v="{{.version}}" -F p={{.packaging}} -F file={{.file}} -u {{.login}}:{{.password}} {{.url}}
EOF
	}]
