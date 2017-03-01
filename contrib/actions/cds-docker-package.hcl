
name = "CDS_DockerPackage"
description = "Build image and push it to docker repository"

// Requirements
requirements = {
	"docker" = {
		type = "binary"
		value = "docker"
	}
	"bash" = {
		type = "binary"
		value = "bash"
	}
}

// Parameters
parameters = {
	 "dockerfileDirectory" = {
		type = "string"
		description = "Directory which contains your Dockerfile."
	}
	"dockerOpts" = {
		type = "string"
		value = ""
		description = "Docker options, Enter --no-cache --pull if you want for example"
	}
	"dockerRegistry" = {
		type = "string"
		description = "Docker Registry. Enter myregistry for build image myregistry/myimage:mytag"
	}
	"imageName" = {
		type = "string"
		description = "Name of your docker image, without tag. Enter myimage for build image myregistry/myimage:mytag"
	}
	"imageTag" = {
		type = "string"
		description = "Tag of your docker image.
Enter mytag for build image myregistry/myimage:mytag. {{.cds.version}} is a good tag from CDS.
You can use many tags: firstTag,SecondTag
Example : {{.cds.version}},latest"
		value = "{{.cds.version}}"
	}
}

// Steps
steps = [{
	script = <<EOF
#!/bin/bash
set -e

IMG=`echo {{.imageName}}| tr '[:upper:]' '[:lower:]'`
GENTAG="cds{{.cds.version}}"
echo "Building {{.dockerRegistry}}/${IMG}:${GENTAG}"

cd {{.dockerfileDirectory}}
docker build {{.dockerOpts}} -t {{.dockerRegistry}}/$IMG:$GENTAG .

IFS=', ' read -r -a tags <<< "{{.imageTag}}"

for t in "${tags[@]}"; do

	set +e

	TAG=`echo ${t} | sed 's/\///g'`
	docker tag {{.dockerRegistry}}/$IMG:$GENTAG {{.dockerRegistry}}/$IMG:$TAG

    echo "Pushing {{.dockerRegistry}}/$IMG:$TAG"
	docker push {{.dockerRegistry}}/$IMG:$TAG

	if [ $? -ne 0 ]; then
		set -e
		echo "/!\ Error while pushing to repository. Automatic retry in 60s..."
	    sleep 60
	    docker push {{.dockerRegistry}}/$IMG:$TAG
	fi

	set -e
	echo " {{.dockerRegistry}}/$IMG:$TAG is pushed"

	docker rmi -f {{.dockerRegistry}}/$IMG:$TAG || true;
done
docker rmi -f {{.dockerRegistry}}/$IMG:$GENTAG || true;

EOF
	}]
