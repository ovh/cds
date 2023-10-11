
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
	--rm \
	-e "GOOS=$(OS)" \
	-e "GOARCH=$(ARCH)" \
	golang:1.21 \
	/bin/bash -c \
	"cd /go/src/github.com/ovh/cds/tests/fixtures/04SCWorkflowRunSimplePlugin && go version && CGO_ENABLED=0 go build -installsuffix cgo -ldflags '-extldflags "-static"' -o plugin-simple-$(OS)-$(ARCH) ."
