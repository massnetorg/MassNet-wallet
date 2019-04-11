# Makefile for MASS

# consts

SCRIPT_BUILD="./scripts/build.sh"
SCRIPT_RELEASE="./scripts/release.sh"
VERSION=`cat ./VER`
TEST_REPORT=test.report

# make commands

builddebug:
	@echo "make build: begin"
	@echo "building mass-wallet to ./bin for current platform..."
	@echo "(To build for all platforms, run 'make buildall')"
	@$(SCRIPT_BUILD) $(VERSION) "CURRENT" "DEBUG"
	@echo "make build: end"

build:
	@echo "make build: begin"
	@echo "building mass-wallet to ./bin for current platform..."
	@echo "(To build for all platforms, run 'make buildall')"
	@$(SCRIPT_BUILD) $(VERSION) "CURRENT"
	@echo "make build: end"

buildall:
	@echo "make buildall: begin"
	@echo "building mass-wallet to ./bin for all platforms..."
	@echo "(To build only for current platform, run 'make build')"
	@$(SCRIPT_BUILD) $(VERSION)
	@echo "make buildall: end"

release:
	@echo "make release: begin"
	@echo "building public-release-version to ./bin/release for all platforms..."
	@$(SCRIPT_RELEASE) $(VERSION)
	@echo "make release: end"

clean:
	@echo "make clean: begin"
	@echo "cleaning .bin/ path..."
	@rm -rf ./bin/*
	@echo "make clean: end"
	@-rm -f $(TEST_REPORT)
