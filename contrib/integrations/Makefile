build: $(TARGET_DIST)
	$(MAKE) build -C arsenal/plugin-arsenal OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C artifactory/plugin-artifactory-build-info OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C artifactory/plugin-artifactory-download-artifact OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C artifactory/plugin-artifactory-promote OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C artifactory/plugin-artifactory-release OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C artifactory/plugin-artifactory-upload-artifact OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C kubernetes/plugin-kubernetes OS="$(OS)" ARCH="$(ARCH)"

clean:
	$(MAKE) clean -C arsenal/plugin-arsenal
	$(MAKE) clean -C artifactory/plugin-artifactory-build-info
	$(MAKE) clean -C artifactory/plugin-artifactory-download-artifact
	$(MAKE) clean -C artifactory/plugin-artifactory-promote
	$(MAKE) clean -C artifactory/plugin-artifactory-release
	$(MAKE) clean -C artifactory/plugin-artifactory-upload-artifact
	$(MAKE) clean -C kubernetes/plugin-kubernetes

ALL_PLUGINS = arsenal/plugin-arsenal artifactory/plugin-artifactory-build-info artifactory/plugin-artifactory-download-artifact artifactory/plugin-artifactory-promote artifactory/plugin-artifactory-release artifactory/plugin-artifactory-upload-artifact kubernetes/plugin-kubernetes
publish: 
	@for P in $(ALL_PLUGINS); do \
		$(MAKE) publish -C $$P; \
	done;
