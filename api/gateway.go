package api

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"

	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	gw "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/config"
)

const (
	// DefaultHTTPLimit default max http conns
	DefaultHTTPLimit = 128
)

func statusUnavailableHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte("{\"err:\",\"Sorry, we received too many simultaneous requests.\nPlease try again later.\"}"))
}

func allowCORS(h http.Handler, config *config.Config) http.Handler {
	httpCh := make(chan bool, DefaultHTTPLimit)
	c := cors.New(cors.Options{
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "HEAD", "POST", "PUT", "DELETE"},
		AllowedOrigins: config.Wallet.API.HttpCORSAddr,
		MaxAge:         600,
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		select {
		case httpCh <- true:
			defer func() { <-httpCh }()
			if len(config.Wallet.API.HttpCORSAddr) == 0 {
				h.ServeHTTP(w, r)
			} else {
				c.Handler(h).ServeHTTP(w, r)
			}
		default:
			statusUnavailableHandler(w, r)
		}
	})
}

func LoadTLSConfig(caPath string) (*tls.Config, error) {
	pool := x509.NewCertPool()
	caCrt, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, err
	}

	if !pool.AppendCertsFromPEM(caCrt) {
		return nil, fmt.Errorf("credentials: failed to append certificates")
	}
	return &tls.Config{
		ClientCAs:  pool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}, nil
}

func Run(cfg *config.Config) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard,
		&runtime.JSONPb{OrigName: true, EmitDefaults: true}))

	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize)),
	}
	err := gw.RegisterApiServiceHandlerFromEndpoint(ctx, mux, "localhost:"+cfg.Wallet.API.GRPCPort, opts)
	if err != nil {
		return err
	}

	handle := maxBytesHandler(mux)
	addr := fmt.Sprintf("%s%s%s", cfg.Wallet.API.Host, ":", cfg.Wallet.API.HttpPort)
	serv := &http.Server{
		Addr:    addr,
		Handler: allowCORS(handle, cfg),
	}

	// http
	if cfg.Wallet.API.DisableTls {
		return serv.ListenAndServe()
	}

	// https
	tlsConfig, err := LoadTLSConfig(cfg.Wallet.API.RpcCert)
	if err != nil {
		return err
	}
	serv.TLSConfig = tlsConfig
	return serv.ListenAndServeTLS(cfg.Wallet.API.RpcCert, cfg.Wallet.API.RpcKey)
}

func maxBytesHandler(h http.Handler) http.Handler {
	const maxReqSize = 1e7 // 10MB
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// A block can easily be bigger than maxReqSize, but everything
		// else should be pretty small.
		req.Body = http.MaxBytesReader(w, req.Body, maxReqSize)
		h.ServeHTTP(w, req)
	})
}
