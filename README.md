# MassNet Wallet

A wallet implementation of [MassNet](http://www.massnet.org/) in Golang.

## Requirements

[Go](http://golang.org) 1.13 or newer.

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

### Contributing Code

#### Prerequisites

- Install [Golang](http://golang.org) 1.13 or newer.
- Install the specific version or [ProtoBuf](https://developers.google.com/protocol-buffers), and related `protoc-*`:
  ```
  # libprotoc
  libprotoc 3.6.1
  
  # github.com/golang/protobuf 1.4.2
  protoc-gen-go
  
  # github.com/gogo/protobuf 1.3.1
  protoc-gen-gogo
  protoc-gen-gofast
  
  # github.com/grpc-ecosystem/grpc-gateway 1.14.5
  protoc-gen-grpc-gateway
  protoc-gen-swagger
  ```

#### Modifying Code

- New codes should be compatible with Go 1.13 or newer.
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