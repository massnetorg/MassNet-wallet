# Makefile for MASS Wallet

# consts
TEST_REPORT=test.report

# make commands
build:
	@echo "make build: begin"
	@echo "building masswallet to ./bin for current platform..."
	@env go build -o bin/masswallet
	@env go build -o bin/masswalletcli cmd/masswalletcli/main.go
	@echo "building masswalletcli to ./bin for current platform..."
	@echo "make build: end"

test:
	@echo "make test: begin"
	@env go test ./... -count=1 -short 2>&1 | tee $(TEST_REPORT)
	@echo "make test: end"
