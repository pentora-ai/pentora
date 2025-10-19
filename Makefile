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

.PHONY: p ps

p: CLI_EXEC=pentora
ps: CLI_EXEC=pentora-server

p ps:
	@PENTORA_CLI_EXECUTABLE=$(CLI_EXEC) go run ./cmd/main.go $(word 2,$(MAKECMDGOALS))

# Parametreler hata vermesin diye
%:
	@:	

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

.PHONY: install-ui-deps
#? install-ui-deps: Install UI dependencies
install-ui-deps:
	cd ui/web && npm install

.PHONY: build-ui
#? build-ui: Build UI for production (outputs to pkg/server/ui/dist)
build-ui: install-ui-deps
	cd ui/web && npm run build
	@echo "✅ UI built successfully → pkg/server/ui/dist"

.PHONY: dev-ui
#? dev-ui: Start UI development server (with API proxy to :8080)
dev-ui:
	cd ui/web && npm run dev

.PHONY: clean-ui
#? clean-ui: Clean UI build artifacts
clean-ui:
	rm -rf pkg/server/ui/dist
	rm -rf ui/web/dist
	@echo "✅ UI build artifacts cleaned"	

.PHONY: generate
#? generate: Generate code (Dynamic and Static configuration documentation reference files)
generate:
#	go generate

.PHONY: binary
#? binary: Build the binary with embedded UI
binary: build-ui generate dist
	@echo "🔨 Building binary with embedded UI..."
	@echo "SHA: $(VERSION) $(CODENAME) $(DATE)"
	CGO_ENABLED=0 GOGC=off GOOS=${GOOS} GOARCH=${GOARCH} go build ${FLAGS[*]} -ldflags "-s -w \
    -X github.com/pentora-ai/pentora/pkg/version.Version=$(VERSION) \
    -X github.com/pentora-ai/pentora/pkg/version.Commit=$(CODENAME) \
    -X github.com/pentora-ai/pentora/pkg/version.BuildDate=$(DATE)" \
    -installsuffix nocgo -o "./dist/${GOOS}/${GOARCH}/$(BIN_NAME)" ./cmd/pentora
	@echo "✅ Binary built → ./dist/${GOOS}/${GOARCH}/$(BIN_NAME)"

.PHONY: lint
#? lint: Run golangci-lint
lint:
	golangci-lint run

.PHONY: validate-files
#? validate-files: Validate code and docs
validate-files:
	$(foreach exec,$(LINT_EXECUTABLES),\
            $(if $(shell which $(exec)),,$(error "No $(exec) in PATH")))
	$(CURDIR)/scripts/validate-vendor.sh
	$(CURDIR)/scripts/validate-misspell.sh
	$(CURDIR)/scripts/validate-shell-script.sh

.PHONY: validate
#? validate: Validate code, docs, and vendor
validate: lint validate-files

.PHONY: help
#? help: Get more info on make commands
help: Makefile
	@echo " Choose a command run in pentora:"
	@sed -n 's/^#?//p' $< | column -t -s ':' |  sort | sed -e 's/^/ /'