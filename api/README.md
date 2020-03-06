# MASS Client API

MASS Client exposes a number of APIs over **gRPC** and **HTTP** for developers to interact with.

[gRPC](https://grpc.io/) is a high-performance open-source RPC framework could be easily implemented across different languages and platforms.

HTTP API provides another clear and universal way to interact by wrapping gRPC. HTTP API's supported MIME content type is `application/json`.

## Usage

### Endpoints

By default, both gPRC and HTTP API will be served at the following ports when MASS Client starts.

| Supported Protocols | Default URL and port     |
|---------------------|--------------------------|
| gRPC                | `http://localhost:50051` |
| HTTP                | `https://localhost:50052` |

### Configuration

| Item           | Default   | Description                                              |
|----------------|-----------|----------------------------------------------------------|
| Host           | `(empty)` | listening host.                                          |
| GRPCPort       | `50051`   | listening gRPC port.                                     |
| HttpPort       | `50052`   | listening HTTP port.                                     |
| HttpCORSAddr   | `(empty)` | Allowed CORS addresses. `*` for allow all.               |

### API Documentation

MASS Client provides a configuration file for [Swagger](https://swagger.io/) which provides a user-friendly HTTP API documentation accessible from web browser.

1. Register an account at [Swagger Hub](https://app.swaggerhub.com).
2. Login and `Import API` from `./api/proto/api.swagger.json`.
3. `Preview Docs`.

Alternatively, check `./api/proto/api.swagger.json` directly for full definition of all RPC APIs.

### Errors

#### Errors in Response Code

| Response Code | Error message                                         |
|---------------|-------------------------------------------------------|
| `400`         | Invalid request.                                      |
| `403`         | User does not have permission to access the resource. |
| `404`         | Resource does not exist.                              |

## Develop

Structure of the `./api/proto` directory,

```c
|- proto/
    | api.proto
    | api.pb.go
    | api.pb.gw.go
    | api.swagger.json
|
```

#### Contributing a new API (en)

- In `./ap/proto/api.proto`:
  - Adding API endpoints in `service.rpc`
  - Adding `Route`, `Method` etc. in `service.rpc.option`
  - Adding structure of request and response in `message`
- In `api_service.go` devlop the handler of the new API. Take care of parameter validation and error handling. New member variables may be added in `server` structure in `server.go`.
- Run the following script to generate `api.pb.go` and `api.pg.gw.go` source codes into `./api/proto`.

```bash
# on Linux-like systems
cd api/proto
./api_gen.sh
# or on Windows
cd api\proto
api_gen.bat
```