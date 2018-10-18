SCALA_VERSION?= 2.11
KAFKA_VERSION?= 0.10.1.1
KAFKA_DIR= kafka_$(SCALA_VERSION)-$(KAFKA_VERSION)
KAFKA_SRC= http://www.mirrorservice.org/sites/ftp.apache.org/kafka/$(KAFKA_VERSION)/$(KAFKA_DIR).tgz
KAFKA_ROOT= testdata/$(KAFKA_DIR)

PKG:=$(shell glide nv)

default: vet test

vet:
	go vet $(PKG)

test: testdeps
	KAFKA_DIR=$(KAFKA_DIR) go test $(PKG) -ginkgo.slowSpecThreshold=60

test-verbose: testdeps
	KAFKA_DIR=$(KAFKA_DIR) go test $(PKG) -ginkgo.slowSpecThreshold=60 -v

test-race: testdeps
	KAFKA_DIR=$(KAFKA_DIR) go test $(PKG) -ginkgo.slowSpecThreshold=60 -v -race

testdeps: $(KAFKA_ROOT)

.PHONY: test testdeps vet

# ---------------------------------------------------------------------

$(KAFKA_ROOT):
	@mkdir -p $(dir $@)
	cd $(dir $@) && curl -sSL $(KAFKA_SRC) | tar xz
