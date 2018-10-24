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

mod:
	@echo "running go mod vendor..." && GO111MODULE=on go mod vendor
	@echo "running go mod tidy... SKIPPED "# && GO111MODULE=on go mod tidy
	@echo "doing some clean in vendor directory..." && find vendor -type f ! \( -name 'modules.txt' -o -name '*.sum' -o -name '*.mod' -o -name '*.rst' -o -name '*.go' -o -name '*.y' -o -name '*.h' -o -name '*.c' -o -name '*.proto' -o -name '*.tmpl' -o -name '*.s' -o -name '*.pl' \) -exec rm {} \;

install:
	@GO111MODULE=on go install -mod=vendor -v $$(go list ./... | grep -v vendor)