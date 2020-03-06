REM API Source Code Generator for MASS Client API Developers
REM Run this script ONLY on Windows
REM See README.md for details

protoc -I=%GOPATH%\src\massnet.org/mass-wallet\api\proto ^
    -I %GOPATH%\src\github.com\grpc-ecosystem\grpc-gateway\third_party\googleapis ^
    -I %GOPATH%\src\github.com\grpc-ecosystem\grpc-gateway ^
    --go_out=plugins=grpc:. ^
    %GOPATH%\src\massnet.org/mass-wallet\api\proto\api.proto

protoc -I=%GOPATH%\src\massnet.org/mass-wallet\api\proto ^
    -I %GOPATH%\src\github.com\grpc-ecosystem\grpc-gateway\third_party\googleapis ^
    -I %GOPATH%\src\github.com\grpc-ecosystem\grpc-gateway ^
    --grpc-gateway_out=logtostderr=true:. ^
    %GOPATH%\src\massnet.org/mass-wallet\api\proto\api.proto

protoc -I=%GOPATH%\src\massnet.org/mass-wallet\api\proto ^
    -I %GOPATH%\src\github.com\grpc-ecosystem\grpc-gateway\third_party\googleapis ^
    -I %GOPATH%\src\github.com\grpc-ecosystem\grpc-gateway ^
    --swagger_out=logtostderr=true:. ^
    %GOPATH%\src\massnet.org/mass-wallet\api\proto\api.proto
