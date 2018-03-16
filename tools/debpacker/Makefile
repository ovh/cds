.PHONY: clean

VERSION := $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot)

TARGET_DIR = ./dist
TARGET_BIN = debpacker
TARGET_LDFLAGS = -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION)"
TARGET_OS = linux
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm 386)

GO_BUILD = go build -v
GO_LINT = ${GOPATH}/bin/gometalinter

$(TARGET_DIR):
	$(info create $(TARGET_DIR) directory)
	@mkdir -p $(TARGET_DIR)

$(GO_LINT):
	go get -u github.com/alecthomas/gometalinter

default: build

clean:
	@rm -rf $(TARGET_DIR)

build: $(TARGET_DIR)
	@for GOOS in $(TARGET_OS); do \
		for GOARCH in $(TARGET_ARCH); do \
			EXTENSION=""; \
			if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
			echo Compiling $(TARGET_DIR)/$(TARGET_BIN)-$$GOOS-$$GOARCH$$EXTENSION $(VERSION); \
			GOOS=$$GOOS GOARCH=$$GOARCH $(GO_BUILD) $(TARGET_LDFLAGS) -o $(TARGET_DIR)/$(TARGET_BIN)-$$GOOS-$$GOARCH$$EXTENSION; \
		done; \
	done


lint: $(GO_LINT)
	$(GO_LINT) --install --force
	$(GO_LINT) --fast --tests --vendor --disable=gas --disable=gotype ./...
