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

install:
	@GO111MODULE=on go install -mod=vendor -v $$(go list ./... | grep -v vendor)