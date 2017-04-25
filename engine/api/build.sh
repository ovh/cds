#!/bin/bash

set -e

BINARY="api"
for GOOS in windows darwin linux freebsd; do
    for GOARCH in amd64 arm; do
        if [[ $GOARCH == "arm" && $GOOS != "linux" ]]; then
          continue;
        fi;
        architecture="${GOOS}-${GOARCH}"

        export GOOS=$GOOS
        export GOARCH=$GOARCH
        outfile="${BINARY}-${architecture}"
        echo "Building ${outfile}"
        go build -ldflags "-X main.VERSION={{.cds.proj.version}}+{{.cds.version}}" -o bin/${outfile}
    done
done
