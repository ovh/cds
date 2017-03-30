
name = "CDS_NexusUpload"
description = "Upload file on Nexus"

// Requirements
requirements = {
	"bash" = {
		type = "binary"
		value = "bash"
	}
<<<<<<< HEAD
	"curl" = {
		type= "binary"
		value = "curl"
	}
}

// Parameters
parameters = {
	"login" = {
		type = "string"
		description = "Login for nexus"
		value = "{{.cds.proj.nexus.login}}"
	}
	"password" = {
		type = "string"
		description = "Password for nexus"
		value = "{{.cds.proj.nexus.password}}"
=======
	"login" = {
		type = "string"
	}
	"password" = {
		type = "string"
>>>>>>> 71710b70412dcb368f25b92164778354dfce7a3f
	}
	"url" = {
		type = "string"
		description = "Nexus URL"
<<<<<<< HEAD
		value = "{{.cds.proj.nexus.url}}"
	}
	"repository" = {
		type = "string"
		description = "Nexus repository that the artifact is contained in"
	}
	"extension" = {
		type = "string"
		description = "Extension of the artifact"
	}
	"groupId" = {
		type = "string"
		description = "Group id of the artifact"
=======
	}
	"repository" = {
		type = "string"
	}
	"extension" = {
		type = "string"
	}
	"groupId" = {
		type = "string"
>>>>>>> 71710b70412dcb368f25b92164778354dfce7a3f
		value = "{{.cds.application}}"
	}
	"artifactId" = {
		type = "string"
<<<<<<< HEAD
		description = "Artifact id of the artifact"
=======
>>>>>>> 71710b70412dcb368f25b92164778354dfce7a3f
		value = "{{.cds.application}}"
	}
	"version" = {
		type = "string"
<<<<<<< HEAD
		description = "Version of the artifact. Supports resolving of "LATEST", "RELEASE" and snapshot versions ("1.0-SNAPSHOT") too."
		value = "{{.cds.build.VERSION}}"
	}
	"files" = {
		type = "string"
		description = "Regex of files you want to upload"
	}
	"packaging" = {
		type = "string"
		description = "Packaging type of the artifact"
	}
}

=======
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

>>>>>>> 71710b70412dcb368f25b92164778354dfce7a3f
// Steps
steps = [{
	script = <<EOF
#!/bin/bash
set -e

<<<<<<< HEAD
echo "Upload to Nexus ({{.url}}) on repository {{.repository}}"

for file in `ls {{.files}}`
do
	if [ -f $file ]
	then
	    curl -F r={{.repository}} -F hasPom=false -F e={{.extension}} -F g="{{.groupId}}" -F a="{{.artifactId}}" -F v="{{.version}}" -F p={{.packaging}} -F file=$file -u {{.login}}:{{.password}} {{.url}}
	else
		echo "File $file does not exist"
	fi
done

=======
echo "action git from directory"
pwd
echo "Upload to Nexus ({{.url}}) on repository {{.repository}}"
curl -F r={{.repository}} -F hasPom=false -F e={{.extension}} -F g="{{.groupId}}" -F a="{{.artifactId}}" -F v="{{.version}}" -F p={{.packaging}} -F file={{.file}} -u {{.login}}:{{.password}} {{.url}}
>>>>>>> 71710b70412dcb368f25b92164778354dfce7a3f
EOF
	}]
