package api

import (
	"flag"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"fmt"

	gw "github.com/massnetorg/MassNet-wallet/api/proto"
)

func Run(portHttp string, portGRPC string) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard,
		&runtime.JSONPb{OrigName: true, EmitDefaults: true}))
	opts := []grpc.DialOption{grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize))}
	echoEndpoint := flag.String("echo_endpoint", ":"+portGRPC, "endpoint of Service")
	err := gw.RegisterApiServiceHandlerFromEndpoint(ctx, mux, *echoEndpoint, opts)
	if err != nil {
		return err
	}

	port := fmt.Sprintf("%s%s", ":", portHttp)
	return http.ListenAndServe(port, mux)
}
