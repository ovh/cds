.PHONY: clean build package copy

TARGET_DIST := $(if ${TARGET_DIST},${TARGET_DIST},./dist)

GO_COV_MERGE = ${GOPATH}/bin/gocovmerge
$(GO_COV_MERGE):
	go install github.com/wadey/gocovmerge@latest

GO_GOJUNIT = ${GOPATH}/bin/go-junit-report
$(GO_GOJUNIT):
	go install github.com/jstemmer/go-junit-report@latest

GO_COBERTURA = ${GOPATH}/bin/gocover-cobertura
$(GO_COBERTURA):
	go install github.com/richardlt/gocover-cobertura@latest

GO_XUTOOLS = ${GOPATH}/bin/xutools
$(GO_XUTOOLS):
	go install github.com/richardlt/xutools@latest

clean:
	$(MAKE) clean -C grpcplugins/action
	$(MAKE) clean -C integrations
	$(MAKE) clean -C vscode-cds
	rm -rf $(TARGET_DIST)

ALL_DISTS := $(shell find . -type d -name "dist" | grep -v node_modules)
ALL_FILES := $(foreach d, $(ALL_DISTS), $(wildcard $(d)/*))

build:
	$(MAKE) build -C grpcplugins/action OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C integrations OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C vscode-cds

$(TARGET_DIST):
	@mkdir -p $(TARGET_DIST)

ALL_TARGETS := $(foreach f, $(ALL_FILES), $(TARGET_DIST)/$(notdir $(f)))

$(ALL_TARGETS): $(TARGET_DIST)
	$(info moving $(filter %/$(notdir $@), $(ALL_FILES)) to $@)
	@mv -f $(filter %/$(notdir $@), $(ALL_FILES)) $@

package: $(ALL_TARGETS) $(TARGET_DIST)

ALL_PLUGINS = $(filter-out null,$(foreach f,$(wildcard $(TARGET_DIST)/*.yml), $(shell cat $(f) | yq .name)))
ALL_ARCHIVES = $(foreach f,$(ALL_PLUGINS),$(strip $(TARGET_DIST)/$(f))-all.tar.gz)

$(ALL_ARCHIVES): ARCHIVE = $(notdir $@)
$(ALL_ARCHIVES): PATTERN = $(subst -all.tar.gz,*[!.tar.gz],$(ARCHIVE))
$(ALL_ARCHIVES):
	$(info Preparing archive $(ARCHIVE) from $(PATTERN))
	cd $(TARGET_DIST); tar czf $(ARCHIVE) $(PATTERN)

archive: $(ALL_ARCHIVES)

clean_archive:
	rm -rf $(TARGET_DIST)/*.tar.gz

test:
	$(MAKE) test -C grpcplugins/action
	$(MAKE) test-coverage
	$(MAKE) test-xunit-report

test-coverage: $(GO_COV_MERGE) $(GO_COBERTURA) $(TARGET_DIST)
	@echo "Generating unit tests coverage..."
	@$(GO_COV_MERGE) `find ./ -name "*.coverprofile"` > $(TARGET_DIST)/cover.out
	@$(GO_COBERTURA) < $(TARGET_DIST)/cover.out > $(TARGET_DIST)/coverage.xml

test-xunit-report: $(GO_GOJUNIT) $(GO_XUTOOLS) $(TARGET_DIST)
	@echo "Generating xUnit Report..."
	@for TST in `find . -name "tests.log"`; do \
		if [ -s $$TST ]; then \
			FAILED=`grep -E '(FAIL)+\s([a-z\.\/]*)\s\[build failed\]' $$TST | wc -l`; \
			if [ $$FAILED -gt 0 ]; then \
				echo "Build Failed \t\t\t($$TST)"; \
				echo "Build Failed \t\t\t($$TST)" >>  $(TARGET_DIST)/fail; \
			else \
				NO_TESTS=`grep -E '\?+\s+([a-z\.\/]*)\s\[no test files\]' $$TST | wc -l`; \
				if [ $$NO_TESTS -gt 0 ]; then \
					echo "No tests found \t\t\t($$TST)"; \
				else \
					if [ "$$TST" = "./tests.log" ]; then \
            PACKAGE=contrib; \
					else \
            PACKAGE=contrib_`echo $$TST | sed 's|./||' | sed 's|/|_|g' | sed 's|_tests.log||'`; \
          fi; \
          XUNIT_OUT=$(TARGET_DIST)/$$PACKAGE.xml; \
          echo "Generating xUnit report from \t$$TST.xml to $$XUNIT_OUT"; \
					cat $$TST | $(GO_GOJUNIT) > $$XUNIT_OUT; \
          TESTS_LOG_OUT=$(TARGET_DIST)/$$PACKAGE.log; \
          echo "Copy tests log from \t$$TST.xml to $$TESTS_LOG_OUT"; \
          cp $$TST $$TESTS_LOG_OUT; \
				fi; \
			fi; \
		else \
			echo "Ignoring empty file \t\t$$TST"; \
		fi; \
	done; \
	$(GO_XUTOOLS) pretty --show-failures ./dist/*.xml > $(TARGET_DIST)/report; \
  $(GO_XUTOOLS) sort-duration ./dist/*.xml > $(TARGET_DIST)/duration; \
	if [ -e $(TARGET_DIST)/report ]; then \
    echo "Report:"; \
		cat $(TARGET_DIST)/report; \
	fi; \
  if [ -e $(TARGET_DIST)/duration ]; then \
    echo "Max duration:"; \
		cat $(TARGET_DIST)/duration; \
	fi; \
	if [ -e $(TARGET_DIST)/fail ]; then \
		echo "#########################"; \
		echo "ERROR: Test compilation failure"; \
		cat $(TARGET_DIST)/fail; \
		exit 1; \
	fi;
