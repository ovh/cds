#!/bin/bash

set -e

for GOOS in windows darwin linux freebsd; do
    for GOARCH in amd64 arm; do
        if [[ $GOARCH == "arm" && $GOOS != "linux" ]]; then
          continue;
        fi;
        architecture="${GOOS}-${GOARCH}"
        echo "Building ${architecture} ${path}"
        export GOOS=$GOOS
        export GOARCH=$GOARCH
        go build -ldflags "-X main.VERSION={{.cds.proj.version}}+{{.cds.version}}" -o bin/cds-${architecture}
    done
done
