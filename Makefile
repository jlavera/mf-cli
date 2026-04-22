.PHONY: install test release

install:
	mise exec -- go build -o out/mf . && sudo cp out/mf /usr/local/bin/mf && sudo xattr -cr /usr/local/bin/mf && sudo codesign --force --sign - /usr/local/bin/mf

test:
	mise exec -- go test ./...

release:
	$(eval LATEST := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"))
	$(eval MAJOR := $(shell echo $(LATEST) | cut -d. -f1))
	$(eval MINOR := $(shell echo $(LATEST) | cut -d. -f2))
	$(eval PATCH := $(shell echo $(LATEST) | cut -d. -f3))
	$(eval NEXT := $(MAJOR).$(MINOR).$(shell echo $$(($(PATCH)+1))))
	$(eval VERSION := $(if $(VERSION),$(VERSION),$(NEXT)))
	@echo "Releasing $(VERSION) (was $(LATEST))"
	git tag $(VERSION)
	git push origin $(VERSION)
