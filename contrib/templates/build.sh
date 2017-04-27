#!/bin/bash

set -e

for template in cds-template-*; do
  [ -d "${template}" ] || continue # if not a directory, skip
  for GOOS in windows darwin linux freebsd; do
      for GOARCH in amd64 arm; do
          if [[ $GOARCH == "arm" && $GOOS != "linux" ]]; then
            continue;
          fi;
          architecture="${GOOS}-${GOARCH}"

          export GOOS=$GOOS
          export GOARCH=$GOARCH
          outfile="${template}-${architecture}"
          echo "Building ${outfile}"
          $(cd ${template} && go build -ldflags "-X main.VERSION={{.cds.proj.version}}+{{.cds.version}}" -o ../bin/${outfile})
      done
  done
done
