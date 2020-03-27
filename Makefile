# Makefile for MASS Wallet

# consts
TEST_REPORT=test.report

# make commands
build:
	@echo "make build: begin"
	@echo "building wallet to ./bin for current platform..."
	@env GO111MODULE=on go build -o bin/masswallet
	@env GO111MODULE=on go build -o bin/masswallet-cli cmd/masswalletcli/main.go
	@echo "make build: end"

test:
	@echo "make test: begin"
	@env GO111MODULE=on go test ./... -count=1 -short 2>&1 | tee $(TEST_REPORT)
	@echo "make test: end"
