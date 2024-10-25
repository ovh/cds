SHELL := /bin/bash
GO_BUILD                 = GOPRIVATE="${GO_PRIVATE}" CGO_ENABLED=0 go build -installsuffix cgo
GO_LIST                  = env GOPRIVATE="${GO_PRIVATE}" go list
TEST_CMD                 = go test -v -timeout 600s -coverprofile=profile.coverprofile
TEST_C_CMD               = go test -c -coverprofile=profile.coverprofile
TEST_RUN_ARGS            = -test.v -test.timeout 600s -test.coverprofile=profile.coverprofile
LDFLAGS                  = -ldflags "$(OPT_LD_FLAGS) -X github.com/ovh/cds/sdk.VERSION=$(VERSION) -X github.com/ovh/cds/sdk.GOOS=$$GOOS -X github.com/ovh/cds/sdk.GOARCH=$$GOARCH -X github.com/ovh/cds/sdk.GITHASH=$(GITHASH) -X github.com/ovh/cds/sdk.BUILDTIME=$(BUILDTIME) -X github.com/ovh/cds/sdk.BINARY=$(TARGET_ENGINE) -X github.com/ovh/cds/sdk.DBMIGRATE=$(DBMIGRATE)"
CURRENT_PACKAGE          = $(shell $(GO_LIST) 2>&1 | grep -v 'no Go files in')
TARGET_DIST              := ./dist
ENABLE_CROSS_COMPILATION := true
GOPATH                   = $(shell go env GOPATH)


##### =====> Clean <===== #####
.PHONY: mk_go_clean
mk_go_clean: # clean target directory
	@rm -rf $(TARGET_DIST)
	@for testfile in `find ./ -name "bin.test"`; do \
		rm $$testfile; \
	done;
	@for TST in `find ./ -name "tests.log"`; do \
		rm $$TST; \
	done;
	@for profile in `find ./ -name "*.coverprofile"`; do \
		rm $$profile; \
	done;

##### =====> Compile <===== #####

IS_TEST                    = $(filter test,$(MAKECMDGOALS))
TARGET_OS                  = $(filter-out $(TARGET_OS_EXCLUDED), $(if ${ENABLE_CROSS_COMPILATION},$(if ${OS},${OS}, $(if $(IS_TEST), $(shell go env GOOS), windows darwin linux openbsd freebsd)),$(shell go env GOOS)))
TARGET_ARCH                = $(if ${ARCH},${ARCH}, $(if $(IS_TEST), $(shell go env GOARCH),amd64 arm 386 arm64 ppc64le))
BINARIES                   = $(addprefix $(TARGET_DIST)/, $(addsuffix -$(OS)-$(ARCH)$(if $(IS_WINDOWS),.exe), $(notdir $(TARGET_NAME))))
OSARCHVALID                := $(shell go tool dist list |grep -v '^darwin/386'|grep -v '^windows/386'|grep -v '^windows/arm'|grep -v '^openbsd/arm*'|grep -v '^openbsd/386'|grep -v '^freebsd/arm*'|grep -v '^freebsd/386')
IS_OS_ARCH_VALID           = $(filter $(OS)/$(ARCH),$(OSARCHVALID))
CROSS_COMPILED_BINARIES    = $(foreach OS, $(TARGET_OS), $(foreach ARCH, $(TARGET_ARCH), $(if $(IS_OS_ARCH_VALID), $(BINARIES))))
GOFILES                    := $(call get_recursive_files, '.')

mk_go_build:
	$(info *** mk_go_build)

$(CROSS_COMPILED_BINARIES): $(GOFILES)
	$(info *** compiling $@)
	@mkdir -p $(TARGET_DIST); \
	GOOS=$(call get_os_from_binary_file,$@) \
	GOARCH=$(call get_arch_from_binary_file,$@) \
	$(GO_BUILD) $(LDFLAGS) -o $@;

##### =====> Compile Tests <===== #####

PKGS     := $(or $(PKG),$(shell $(GO_LIST) ./...))
TESTPKGS := $(shell $(GO_LIST) -f \
			'{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' \
			$(PKGS) 2>&1 | grep -v 'no Go files in' )

TESTPKGS_C_FILE = $(addsuffix /bin.test, $(subst $(CURRENT_PACKAGE),.,$(PKG)))
TESTPKGS_C = $(foreach PKG, $(TESTPKGS), $(TESTPKGS_C_FILE))

$(TESTPKGS_C): #main_test.go
	$(info *** compiling test $@)
	@cd $(dir $@) && $(TEST_C_CMD) -o bin.test .

##### =====> Running Tests <===== #####

TESTPKGS_RESULTS_LOG_FILE = $(addsuffix /tests.log, $(subst $(CURRENT_PACKAGE),.,$(PKG)))
TESTPKGS_RESULTS = $(foreach PKG, $(TESTPKGS), $(TESTPKGS_RESULTS_LOG_FILE))

$(HOME)/.richstyle.yml:
	echo "leaveTestPrefix: true" > $(HOME)/.richstyle.yml

GO_RICHGO = $(GOPATH)/bin/richgo
$(GO_RICHGO): $(HOME)/.richstyle.yml
	go install github.com/kyoh86/richgo@latest

EXIT_TESTS := 0
$(TESTPKGS_RESULTS): $(GOFILES) $(TESTPKGS_C) $(GO_RICHGO)
	$(info *** executing tests in $(dir $@))
	@-cd $(dir $@) && ./bin.test $(TEST_RUN_ARGS) | tee tests.log | richgo testfilter ;

mk_go_test: $(GOFILES) $(TESTPKGS_RESULTS) # Run tests

##### =====> lint <===== #####

GOLANG_CI_LINT 		:= $(GOPATH)/bin/golangci-lint
$(GOLANG_CI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.27.0

mk_go_lint: $(GOLANG_CI_LINT) # run golangci lint
	$(info *** running lint)
	$(GOLANG_CI_LINT) run

##### =====> Internals <===== #####

$(TARGET_DIST):
	$(info create $(TARGET_DIST) directory)
	@mkdir -p $(TARGET_DIST)

define get_os_from_binary_file
$(strip $(shell echo $(1) | cut -d'_' -f 2))
endef

define get_arch_from_binary_file
$(strip $(patsubst %.exe, %,$(shell echo $(1) | cut -d'_' -f 3)))
endef

define get_recursive_files
$(subst ./,,$(shell find $(1) -type f -name "*.go" -print))
endef
