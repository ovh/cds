UNAME := $(shell uname)
SHA512 := $(if ifeq ${UNAME} "Darwin",shasum -a 512,sha512sum)
VERSION := $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot)

clean:
	@rm -rf dist
	@rm -rf node_modules
	@rm -rf semantic
	@rm -rf semantic\\
	@rm -f package-lock.json

NG = node_modules/@angular/cli/bin/ng
$(NG):
	npm install

stats: $(NG)
	@node --max-old-space-size=2048 node_modules/@angular/cli/bin/ng build --prod --stats-json

ngbuild: $(NG)
	@node --max-old-space-size=2048 node_modules/@angular/cli/bin/ng build --prod

INDEX = dist/index.tmpl
$(INDEX):
	$(MAKE) ngbuild
	@cd dist && mv index.html index.tmpl

FILES_UI = dist/FILES_UI
$(FILES_UI): $(INDEX)
	$(info sha512 = ${SHA512})
	touch $(FILES_UI)
	cd dist/ && for i in `ls -p | grep -v /|grep -v FILES_UI`; do echo "$$i;`${SHA512} $$i|cut -d ' ' -f1`" >> FILES_UI; done;

build: $(FILES_UI) $(INDEX) ui.tar.gz

ui.tar.gz:
	tar cfz ui.tar.gz dist

lintfix:
	./node_modules/.bin/ng lint --fix
