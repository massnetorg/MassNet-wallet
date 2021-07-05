# MassNet Wallet

  A wallet implementation of [MassNet](http://www.massnet.org/) in Golang.

## Requirements

  [Go](http://golang.org) 1.11 or newer.

## Development

### Build from Source

#### Linux/Darwin

- Clone source code to `$GOPATH/src/massnet.org`.
- Build the program.
  ```bash
  cd $GOPATH/src/massnet.org/mass-wallet
  make build
  ```
- Run `./bin/masswallet` to start.

#### Windows

- Clone source code to `$GOPATH\src\massnet.org`.
- Set environment variable `GO111MODULE="off"` if your Golang version is newer than 1.11.
- Build the program.
  ```bat
  cd %GOPATH%\src\massnet.org\mass-wallet
  go build -o bin\masswallet.exe
  ```
- Run `bin\masswallet.exe` to start.

### Contributing Code

#### Prerequisites

- Install [Golang](http://golang.org) 1.11 or newer.
- Install the specific version or [ProtoBuf](https://developers.google.com/protocol-buffers), and related `protoc-*`:
  ```
  # libprotoc
  libprotoc 3.6.1
  
  # github.com/golang/protobuf 1.3.2
  protoc-gen-go
  
  # github.com/gogo/protobuf 1.2.1
  protoc-gen-gogo
  protoc-gen-gofast
  
  # github.com/grpc-ecosystem/grpc-gateway 1.9.6
  protoc-gen-grpc-gateway
  protoc-gen-swagger
  ```

#### Modifying Code

- New codes should be compatible with Go 1.11 or newer.
- Run `gofmt` and `goimports` to lint go files.
- Run `make test` before building executables.

#### Reporting Bugs

Contact MASS community via community@massnet.org, and we will get back to you as soon as possible.

## Documentation

### API

A document for API is provided [here](docs/API_en.md).

### Transaction Scripts

A document for Transaction Scripts is provided [here](docs/script_en.md).

## License

`MassNet Wallet` is licensed under the terms of the MIT license. See LICENSE for more information or see https://opensource.org/licenses/MIT.