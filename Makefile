.PHONY: clean doc modclean mod goinstall build dist deb

CDS_VERSION := $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot)
GIT_VERSION := $(if ${GIT_DESCRIBE},${GIT_DESCRIBE},$(shell git describe))

TARGET_ENGINE = engine
TARGET_WORKER = worker
TARGET_CDSCTL = cdsctl

doc:
ifndef GEN_PATH
	$(error GEN_PATH is undefined)
endif
    # export GEN_PATH=$HOME/src/github.com/ovh/cds/docs/content/docs/components
	$(TARGET_CDSCTL) doc $(GEN_PATH)
	$(TARGET_WORKER) doc $(GEN_PATH)
	$(TARGET_ENGINE) doc $(GEN_PATH) ./
	cd docs && ./build.sh

modclean:
	@echo "removing vendor directory... " && rm -rf vendor
	@echo "cleaning modcache... " && GO111MODULE=off go clean -modcache || true

mod:
	@echo "running go mod tidy... " && GO111MODULE=on go mod tidy
	@echo "running go mod vendor..." && GO111MODULE=on go mod vendor
	@echo "doing some clean in vendor directory..." && find vendor -type f ! \( -name 'modules.txt' -o -name '*.sum' -o -name '*.mod' -o -name '*.rst' -o -name '*.go' -o -name '*.y' -o -name '*.h' -o -name '*.c' -o -name '*.proto' -o -name '*.tmpl' -o -name '*.s' -o -name '*.pl' \) -exec rm {} \;
	# two calls to RegisterManifestSchema(ocispec.MediaTypeImageIndex -> panic
	# file oci.go is in conflict with file /vendor/github.com/docker/distribution/manifest/manifestlist/manifestlist.go
	# when docker update their vendor, it will be possible to remove this line.
	# this will fix the plugin-clair for the moment
	@echo "removing file /vendor/github.com/docker/docker/distribution/oci.go..." && rm -f vendor/github.com/docker/docker/distribution/oci.go
	@echo "removing subpackages vendors" &&  rm -rf vendor/github.com/ovh/cds

goinstall:
	go install $$(go list ./...)

build:
	$(MAKE) build -C ui
	$(MAKE) build -C engine
	$(MAKE) build -C engine/worker
	$(MAKE) build -C cli/cdsctl

ENGINE_DIST = $(wildcard engine/dist/*)
WORKER_DIST = $(wildcard engine/worker/dist/*)
CLI_DIST = $(wildcard cli/cdsctl/dist/*)
UI_DIST = ui/ui.tar.gz

TARGET_DIR := dist/
ALL_DIST = $(ENGINE_DIST) 
ALL_DIST := $(ALL_DIST) $(WORKER_DIST) 
ALL_DIST := $(ALL_DIST) $(CLI_DIST) 
ALL_DIST := $(ALL_DIST) $(UI_DIST)
ALL_TARGETS := $(foreach DIST,$(ALL_DIST),$(addprefix $(TARGET_DIR),$(notdir $(DIST))))

define get_dist_from_target
$(filter %/$(notdir $(1)), $(ALL_DIST))
endef

$(ALL_TARGETS):
	@mkdir -p $(TARGET_DIR)
	$(info copying $(call get_dist_from_target, $@) to $@) 
	@cp -f $(call get_dist_from_target, $@) $@

dist: $(ALL_TARGETS)

clean: 
	@rm -rf target
	$(MAKE) clean -C engine
	$(MAKE) clean -C engine/worker
	$(MAKE) clean -C cli/cdsctl
	$(MAKE) clean -C ui

deb: $(ALL_TARGETS) target/cds-engine.deb
	
$(TARGET_DIR)/config.toml.sample:
	$(TARGET_DIR)/cds-engine-linux-amd64 config new > $(TARGET_DIR)/config.toml.sample

target/cds-engine.deb: $(TARGET_DIR)/config.toml.sample
	debpacker make --workdir dist --config .debpacker.yml --var version=$(GIT_VERSION)