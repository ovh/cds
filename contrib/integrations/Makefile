build: $(TARGET_DIST)
	$(MAKE) build -C arsenal/plugin-arsenal OS=$(OS) ARCH=$(ARCH)
	$(MAKE) build -C kubernetes/plugin-kubernetes OS=$(OS) ARCH=$(ARCH)

clean:
	$(MAKE) clean -C arsenal/plugin-arsenal
	$(MAKE) clean -C kubernetes/plugin-kubernetes