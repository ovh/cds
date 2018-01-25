#!/bin/bash

for GOOS in windows darwin linux freebsd; do
    for GOARCH in 386 amd64 arm; do
        if [[ $GOARCH == "arm" && $GOOS != "linux" ]]; then
          continue;
        fi;
        architecture="${GOOS}-${GOARCH}"
        echo "Building ${architecture} ${path}"
        export GOOS=$GOOS
        export GOARCH=$GOARCH
        go build -ldflags "-X ${PROJECT_PATH}/${PROJECT_NAME}/update.architecture=${architecture} -X ${PROJECT_PATH}/${PROJECT_NAME}/update.urlUpdateSnapshot=${URL_UPDATE_SNAPSHOT}" -o bin/tatcli-${architecture}
    done
done
