# consts
SCRIPT_BUILD="./build.sh"

VERSION=`cat ../../VER`

.PHONY:dev

# make commands

builddebug:
	@echo "make build: begin"
	@echo "building masscli to ./bin for current platform..."
	@echo "(To build for all platforms, run 'make buildall')"
	@$(SCRIPT_BUILD) $(VERSION) "CURRENT" "DEBUG"
	@echo "make build: end"

build:
	@echo "make build: begin"
	@echo "building masscli to ./bin for current platform..."
	@echo "(To build for all platforms, run 'make buildall')"
	@$(SCRIPT_BUILD) $(VERSION) "CURRENT"
	@echo "make build: end"

buildall:
	@echo "make buildall: begin"
	@echo "building masscli to ./bin for all platforms..."
	@echo "(To build only for current platform, run 'make build')"
	@$(SCRIPT_BUILD) $(VERSION)
	@echo "make buildall: end"

clean:
	@echo "make clean: begin"
	@echo "cleaning .bin/ path..."
	@rm -rf ./bin/*
	@echo "make clean: end"
