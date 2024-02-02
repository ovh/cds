DIST := dist

.PHONY: clean package

$(DIST):
	mkdir -p $(DIST)

build: $(DIST)
	npm install --include=dev
	npm run vsce package
	mv vscode-cds-*.vsix $(DIST)

package: $(TARGET_DIST) build
	cp -f $(DIST)/vscode-cds-*.vsix $(TARGET_DIST)

clean:
	rm -f $(DIST)/vscode-cds-*.vsix
