
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
		description = "Docker Registry Url. Enter myregistry url for build image myregistry/myimage:mytag"
	}
	"dockerRegistryUsername" = {
		type = "string"
		description = "Docker Registry Username. Enter username to connect on your docker registry."
	}
	"dockerRegistryPassword" = {
		type = "string"
		description = "Docker Registry Password. Enter password to connect on your docker registry."
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
REGISTRY="{{.dockerRegistry}}"
USERNAME="{{.dockerRegistryUsername}}"
PASSWORD="{{.dockerRegistryPassword}}"

echo "Building ${REGISTRY}/${IMG}:${GENTAG}"

cd {{.dockerfileDirectory}}
docker build {{.dockerOpts}} -t ${REGISTRY}/${IMG}:${GENTAG} .

IFS=', ' read -r -a tags <<< "{{.imageTag}}"

for t in "${tags[@]}"; do

	set +e

	if [[ ! -z "${USERNAME}" && ! -z "${PASSWORD}" && ! -z "${REGISTRY}" ]]; then
		echo "Login to ${REGISTRY}"
		docker login -u ${USERNAME} -p ${PASSWORD} ${REGISTRY}
	fi

	TAG=`echo ${t} | sed 's/\///g'`
	docker tag ${REGISTRY}/${IMG}:${GENTAG} ${REGISTRY}/${IMG}:${TAG}

  echo "Pushing ${REGISTRY}/${IMG}:${TAG}"
	docker push ${REGISTRY}/${IMG}:${TAG}

	if [ $? -ne 0 ]; then
		set -e
		echo "/!\ Error while pushing to repository. Automatic retry in 60s..."
		sleep 60
		docker push ${REGISTRY}/${IMG}:${TAG}
	fi

	set -e
	echo " ${REGISTRY}/${IMG}:${TAG} is pushed"

	docker rmi -f ${REGISTRY}/${IMG}:${TAG} || true;
done

IMAGE_ID=`docker images --digests --no-trunc --format "{{.Repository}}:{{.Tag}} {{.ID}}" | grep "${REGISTRY}/${IMG}:${GENTAG}" | awk '{print $2}'`
IMAGE_DIGEST=`docker images --digests --no-trunc --format "{{.Repository}}:{{.Tag}} {{.Digest}}" | grep "${REGISTRY}/${IMG}:${GENTAG}" | awk '{print $2}'`

echo "ID=$IMAGE_ID"
worker export image.id ${IMAGE_ID}

echo "DIGEST=$IMAGE_DIGEST"
worker export image.digest ${IMAGE_DIGEST}

docker rmi -f ${REGISTRY}/${IMG}:${GENTAG} || true;

EOF
	}]
