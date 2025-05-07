# Makefile for macOS .pkg + .dmg installer

SRCS = $(shell git ls-files '*.go' | grep -v '^vendor/')

APP_NAME=pentora
VERSION=1.0.0
IDENTIFIER=com.pentora.cli
PKG_ROOT=pkgroot
BUILD_DIR=build
DMG_ROOT=dmgroot

# Default build target
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

LINT_EXECUTABLES = misspell shellcheck

.PHONY: default
#? default: Run `make generate` and `make binary`
default: generate binary

.PHONY: test
#? test: Run the unit and integration tests
test: test-unit

.PHONY: test-unit
#? test-unit: Run the unit tests
test-unit:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go test -cover "-coverprofile=cover.out" -v $(TESTFLAGS) ./pkg/... ./cmd/...	

.PHONY: fmt
#? fmt: Format the Code
fmt:
	gofmt -s -l -w $(SRCS)	

#? dist: Create the "dist" directory
dist:
	mkdir -p dist

.PHONY: build-ui-image
#? build-ui-image: Build UI Docker image
build-ui-image:
	docker build -t pentora-ui -f ui/Dockerfile ui	

.PHONY: clean-ui
#? clean-ui: Clean UI static generated assets
clean-ui:
	rm -r ui/static
	mkdir -p ui/static
	printf 'For more information see `ui/readme.md`' > ui/static/DONT-EDIT-FILES-IN-THIS-DIRECTORY.md	

ui/static/index.html:
	$(MAKE) build-ui-image
	docker run --rm -v "$(PWD)/ui/static":'/src/ui/static' pentora-ui npm run build:nc
	docker run --rm -v "$(PWD)/ui/static":'/src/ui/static' pentora-ui chown -R $(shell id -u):$(shell id -g) ./static

.PHONY: generate-ui
#? generate-ui: Generate UI
generate-ui: ui/static/index.html	

.PHONY: generate
#? generate: Generate code (Dynamic and Static configuration documentation reference files)
generate:
#	go generate

.PHONY: binary
#? binary: Build the binary
binary: generate-ui dist
	@echo SHA: $(VERSION) $(CODENAME) $(DATE)
	CGO_ENABLED=0 GOGC=off GOOS=${GOOS} GOARCH=${GOARCH} go build ${FLAGS[*]} -ldflags "-s -w \
    -X github.com/pentoraai/pentora/pkg/version.Version=$(VERSION) \
    -X github.com/pentoraai/pentora/pkg/version.Codename=$(CODENAME) \
    -X github.com/pentoraai/pentora/pkg/version.BuildDate=$(DATE)" \
    -installsuffix nocgo -o "./dist/${GOOS}/${GOARCH}/$(BIN_NAME)" ./cmd/pentora

.PHONY: lint
#? lint: Run golangci-lint
lint:
	golangci-lint run

.PHONY: validate-files
#? validate-files: Validate code and docs
validate-files:
	$(foreach exec,$(LINT_EXECUTABLES),\
            $(if $(shell which $(exec)),,$(error "No $(exec) in PATH")))
	$(CURDIR)/script/validate-vendor.sh
	$(CURDIR)/script/validate-misspell.sh
	$(CURDIR)/script/validate-shell-script.sh

.PHONY: validate
#? validate: Validate code, docs, and vendor
validate: lint validate-files

.PHONY: help
#? help: Get more info on make commands
help: Makefile
	@echo " Choose a command run in pentora:"
	@sed -n 's/^#?//p' $< | column -t -s ':' |  sort | sed -e 's/^/ /'