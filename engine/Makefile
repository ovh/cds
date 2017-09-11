.PHONY: build test lint

ldflags = -ldflags "-X github.com/ovh/cds/sdk.VERSION=snapshot"

default: build

build:
	go build $(ldflags)

test:
	go get -u github.com/wadey/gocovmerge
	go get -u github.com/mattn/goveralls
	for PKG in $(go list ./... | grep -v vendor); \
	do \
		cd ${GOPATH}/src/${PKG} ;
		go test -v -timeout 180s -coverprofile=profile.coverprofile | tee ${GOPATH}/src/${PKG}/tests.log ; \
		if [ $? -ne 0 ] ; then \
			export EXIT_CODE=1 ; \
		fi ; \
	done
	cd ${GOPATH}/src/github.com/ovh/cds/engine
	gocovmerge ./**/*.coverprofile > cover.out
	go tool cover -html=cover.out -o=cover.html

lint:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install --force
	gometalinter --fast --tests --vendor --disable=gas --disable=gotype ./...