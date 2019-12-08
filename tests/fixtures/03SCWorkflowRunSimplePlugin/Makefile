
build: plugin-simple-linux-amd64 plugin-simple-darwin-amd64

clean:
	@rm plugin-simple-linux-amd64 plugin-simple-darwin-amd64

plugin-simple-linux-amd64:
	$(MAKE) build-plugin OS=linux ARCH=amd64

plugin-simple-darwin-amd64:
	$(MAKE) build-plugin OS=darwin ARCH=amd64

build-plugin:
	@docker run \
	--mount type=bind,source=$$(pwd)/../../../,dst=/go/src/github.com/ovh/cds \
	--mount type=bind,source=$$(pwd),dst=/go/src/plugin/build \
	--rm \
	-e "GO111MODULE=off" \
	-e "GOOS=$(OS)" \
	-e "GOARCH=$(ARCH)" \
	golang:1.13 \
	/bin/bash -c \
	"cd /go/src/plugin/build && go version &&  go get && CGO_ENABLED=0 go build -installsuffix cgo -ldflags '-extldflags "-static"' -o plugin-simple-$(OS)-$(ARCH) ."
