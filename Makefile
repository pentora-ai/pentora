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

.PHONY: all clean build pkg dmg

all: clean build pkg dmg

build:
	@echo "> Building $(APP_NAME) binary for macOS"
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/pentora

pkg: build
	@echo "> Preparing package root"
	mkdir -p $(PKG_ROOT)/usr/local/bin
	cp $(BUILD_DIR)/$(APP_NAME) $(PKG_ROOT)/usr/local/bin/$(APP_NAME)
	@echo "> Creating .pkg installer"
	pkgbuild \
	  --identifier $(IDENTIFIER) \
	  --version $(VERSION) \
	  --root $(PKG_ROOT) \
	  --install-location / \
	  $(BUILD_DIR)/$(APP_NAME).pkg

dmg: pkg
	@echo "> Creating .dmg image"
	@mkdir -p $(DMG_ROOT)
	@cp $(BUILD_DIR)/$(APP_NAME).pkg $(DMG_ROOT)/
	create-dmg \
	  --volname "Pentora Installer" \
	  --window-size 500 300 \
	  --icon-size 100 \
	  --icon "$(APP_NAME).pkg" 200 120 \
	  --app-drop-link 400 120 \
	  $(BUILD_DIR)/$(APP_NAME)-installer.dmg \
	  $(DMG_ROOT)

clean:
	rm -rf $(BUILD_DIR) $(PKG_ROOT) $(DMG_ROOT)


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
