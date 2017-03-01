#!/bin/bash

# tat
cd api
for GOOS in darwin linux ; do
    GOARCH=amd64
    architecture="${GOOS}-${GOARCH}"
    echo "Building ${architecture} ${path}"
    export GOOS=$GOOS
    export GOARCH=$GOARCH
    CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/tat-${architecture}
    echo "file bin/tat-${architecture}"
    file bin/tat-${architecture}
    echo "ldd bin/tat-${architecture}"
done

# tatcli
cd ../tatcli
for GOOS in windows darwin linux freebsd; do
    for GOARCH in 386 amd64 arm; do
        if [[ $GOARCH == "arm" && $GOOS != "linux" ]]; then
          continue;
        fi;
        architecture="${GOOS}-${GOARCH}"
        echo "Building ${architecture} ${path}"
        export GOOS=$GOOS
        export GOARCH=$GOARCH
        go build -ldflags "-X github.com/ovh/tat/tatcli/update.architecture=${architecture}" -o bin/tatcli-${architecture}
    done
done
