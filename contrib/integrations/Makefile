build: $(TARGET_DIST)
	$(MAKE) build -C arsenal/arsenal-deployment-plugin OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C artifactory/artifactory-build-info-plugin OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C artifactory/artifactory-download-artifact-plugin OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C artifactory/artifactory-promote-plugin OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C artifactory/artifactory-release-plugin OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C artifactory/artifactory-upload-artifact-plugin OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C kubernetes/plugin-kubernetes-deployment OS="$(OS)" ARCH="$(ARCH)"

clean:
	$(MAKE) clean -C arsenal/arsenal-deployment-plugin
	$(MAKE) clean -C artifactory/artifactory-build-info-plugin
	$(MAKE) clean -C artifactory/artifactory-download-artifact-plugin
	$(MAKE) clean -C artifactory/artifactory-promote-plugin
	$(MAKE) clean -C artifactory/artifactory-release-plugin
	$(MAKE) clean -C artifactory/artifactory-upload-artifact-plugin
	$(MAKE) clean -C kubernetes/plugin-kubernetes-deployment

ALL_PLUGINS = arsenal/arsenal-deployment-plugin artifactory/artifactory-build-info-plugin artifactory/artifactory-download-artifact-plugin artifactory/artifactory-promote-plugin artifactory/artifactory-release-plugin artifactory/artifactory-upload-artifact-plugin kubernetes/plugin-kubernetes-deployment
publish:
	@for P in $(ALL_PLUGINS); do \
		$(MAKE) publish -C $$P; \
	done;
