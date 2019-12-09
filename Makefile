.PHONY: clean doc modclean mod goinstall build dist deb

TARGET_OS = $(if ${OS},${OS},windows darwin linux freebsd)
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm 386 arm64)
CDS_VERSION := $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot)
GIT_DESCRIBE := $(shell git describe)
GIT_VERSION := $(if ${GIT_DESCRIBE},${GIT_DESCRIBE},0.42.0-99-snapshot) #TODO fixme

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
	@echo "cleaning modcache... " && GO111MODULE=off go clean -modcache || true

mod:
	@echo "running go mod tidy... " && GO111MODULE=on go mod tidy


ENGINE_DIST = $(wildcard engine/dist/*)
WORKER_DIST = $(wildcard engine/worker/dist/*)
CLI_DIST = $(wildcard cli/cdsctl/dist/*)
CONTRIB_DIST = $(wildcard contrib/dist/*)
UI_DIST = ui/ui.tar.gz

TARGET_DIR := dist/
ALL_DIST = $(ENGINE_DIST) 
ALL_DIST := $(ALL_DIST) $(WORKER_DIST) 
ALL_DIST := $(ALL_DIST) $(CLI_DIST) 
ALL_DIST := $(ALL_DIST) $(UI_DIST)
ALL_DIST := $(ALL_DIST) $(CONTRIB_DIST)
ALL_TARGETS := $(foreach DIST,$(ALL_DIST),$(addprefix $(TARGET_DIR),$(notdir $(DIST))))


goinstall:
	go install $$(go list ./...)

build:
	$(info Building CDS Components for $(TARGET_OS) - $(TARGET_ARCH))
	$(MAKE) build -C ui
	$(MAKE) build -C engine OS="${TARGET_OS}" ARCH="${TARGET_ARCH}"
	$(MAKE) build -C engine/worker OS="${TARGET_OS}" ARCH="${TARGET_ARCH}"
	$(MAKE) build -C cli/cdsctl OS="${TARGET_OS}" ARCH="${TARGET_ARCH}"
	$(MAKE) build -C contrib OS="${TARGET_OS}" ARCH="${TARGET_ARCH}"
	$(MAKE) package -C contrib TARGET_DIST="$(abspath $(TARGET_DIR))"

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
	$(MAKE) clean -C contrib

deb: $(ALL_TARGETS) target/cds-engine.deb
	
$(TARGET_DIR)/config.toml.sample:
	$(TARGET_DIR)/cds-engine-linux-amd64 config new > $(TARGET_DIR)/config.toml.sample

target/cds-engine.deb: $(TARGET_DIR)/config.toml.sample
	debpacker make --workdir dist --config .debpacker.yml --var version=$(GIT_VERSION)
