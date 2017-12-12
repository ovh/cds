#!/bin/bash

cd cli/venom
for DIST in `go tool dist list | grep -v '^android/' | grep -v '^nacl/' | grep -v '^plan9/' | grep -v '^darwin/arm'`; do
    GOOS=`echo ${DIST} | cut -d / -f 1`
    GOARCH=`echo ${DIST} | cut -d / -f 2`

    architecture="${GOOS}-${GOARCH}"
    echo "Building ${architecture} ${path}"
    export GOOS=$GOOS
    export GOARCH=$GOARCH
    go build -ldflags "-X github.com/ovh/venom/cli/venom/update.architecture=${architecture}" -o bin/venom.${architecture}
done
