TARGET_ENGINE = engine
TARGET_WORKER = worker
TARGET_CDSCTL = cdsctl

doc:
ifndef GEN_PATH
	$(error GEN_PATH is undefined)
endif
	$(TARGET_CDSCTL) doc $(GEN_PATH)
	$(TARGET_WORKER) doc $(GEN_PATH)
	$(TARGET_ENGINE) doc $(GEN_PATH) ./
	cd docs && ./build.sh

modclean:
	@echo "removing vendor directory... " && rm -rf vendor
	@echo "cleaning modcache... " && GO111MODULE=on go clean -modcache || true

mod:
	@echo "running go mod tidy... " && GO111MODULE=on go mod tidy
	@echo "running go mod vendor..." && GO111MODULE=on go mod vendor
	@echo "doing some clean in vendor directory..." && find vendor -type f ! \( -name 'modules.txt' -o -name '*.sum' -o -name '*.mod' -o -name '*.rst' -o -name '*.go' -o -name '*.y' -o -name '*.h' -o -name '*.c' -o -name '*.proto' -o -name '*.tmpl' -o -name '*.s' -o -name '*.pl' \) -exec rm {} \;
	# two calls to RegisterManifestSchema(ocispec.MediaTypeImageIndex -> panic
	# file oci.go is in conflict with file /vendor/github.com/docker/distribution/manifest/manifestlist/manifestlist.go
	# when docker update their vendor, it will be possible to remove this line.
	# this will fix the plugin-clair for the moment
	@echo "removing file /vendor/github.com/docker/docker/distribution/oci.go..." && rm -f vendor/github.com/docker/docker/distribution/oci.go

install:
	go install -v $$(go list ./... | grep -v vendor)
