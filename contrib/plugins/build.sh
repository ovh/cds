#!/bin/bash

set -e

for plugin in plugin-*; do
  [ -d "${plugin}" ] || continue # if not a directory, skip
  for GOOS in windows darwin linux freebsd; do
      for GOARCH in amd64 arm; do
          if [[ $GOARCH == "arm" && $GOOS != "linux" ]]; then
            continue;
          fi;
          architecture="${GOOS}-${GOARCH}"

          export GOOS=$GOOS
          export GOARCH=$GOARCH
          outfile="${plugin}-${architecture}"
          echo "Building ${outfile}"
          $(cd ${plugin} && go build -ldflags "-X main.VERSION={{.cds.proj.version}}+{{.cds.version}}" -o ../bin/${outfile})
      done
  done
done
