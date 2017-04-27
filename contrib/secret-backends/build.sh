#!/bin/bash

set -e

for secret in secret-*; do
  [ -d "${secret}" ] || continue # if not a directory, skip
  for GOOS in windows darwin linux freebsd; do
      for GOARCH in amd64 arm; do
          if [[ $GOARCH == "arm" && $GOOS != "linux" ]]; then
            continue;
          fi;
          architecture="${GOOS}-${GOARCH}"

          export GOOS=$GOOS
          export GOARCH=$GOARCH
          outfile="${secret}-${architecture}"
          echo "Building ${outfile}"
          $(cd ${secret} && go build -ldflags "-X main.VERSION={{.cds.proj.version}}+{{.cds.version}}" -o ../bin/${outfile})
      done
  done
done
