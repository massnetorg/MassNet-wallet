package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/massnetorg/mass-core/logging"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	jww "github.com/spf13/jwalterweatherman"
	spb "google.golang.org/genproto/googleapis/rpc/status"
)

// server is the interface for http/rpc server.
type server interface {
	call(ctx context.Context, u *url.URL, method Method, bodyReader io.Reader) (io.ReadCloser, error)
}

type Method uint32

var methodRef = map[Method]string{GET: "GET", POST: "POST", PUT: "PUT", DELETE: "DELETE"}

func (m Method) String() string {
	if method, exists := methodRef[m]; exists {
		return method
	}
	return "INVALID"
}

const (
	// GET represents GET method in http
	GET Method = iota
	// POST represents POST method in http
	POST
	// PUT represents PUT method in http
	PUT
	// DELETE represents DELETE method in http
	DELETE
)

// Client is the top level for http/rpc server
type Client struct {
	url    *url.URL
	client *http.Client
}

var client = &Client{}

func initClient() {
	u, err := url.Parse(config.Server)
	if err != nil {
		logging.VPrint(logging.FATAL, "failed to parse url", logging.LogFormat{"err": err})
	}
	client.url = u

	switch {
	case u.Scheme == "https":
		cert, err := tls.LoadX509KeyPair(config.RpcCert, config.RpcKey)
		if err != nil {
			logging.VPrint(logging.FATAL, "failed to load certificate", logging.LogFormat{"err": err})
		}

		client.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
					Certificates:       []tls.Certificate{cert},
				},
			},
		}
	case u.Scheme == "http":
		client.client = http.DefaultClient
	default:
		logging.VPrint(logging.FATAL, "unsupported scheme", logging.LogFormat{"scheme": u.Scheme})
	}
}

func (c *Client) call(ctx context.Context, u *url.URL, method Method, bodyReader io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method.String(), u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil && ctx.Err() != nil { // check if it timed out
		return nil, ctx.Err()
	} else if err != nil {
		return nil, err
	}

	return resp, nil
}

// CallRaw calls a remote node, specified by the path.
// It returns the raw response body
func (c *Client) CallRaw(ctx context.Context, path string, method Method, request interface{}) (*http.Response, error) {
	c.url.Path = path

	var bodyReader io.Reader
	if request != nil {
		var jsonBody bytes.Buffer
		m := jsonpb.Marshaler{EmitDefaults: false}
		m.Marshal(&jsonBody, request.(proto.Message))
		bodyReader = &jsonBody
	}

	return c.call(ctx, c.url, method, bodyReader)
}

// Call calls a remote node, specified by the path.
func (c *Client) Call(ctx context.Context, path string, method Method, request, response interface{}) error {
	resp, err := c.CallRaw(ctx, path, method, request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %v", err)
		}

		var st spb.Status
		if err = json.Unmarshal(buf, &st); err != nil {
			return fmt.Errorf("failed to unmarshal response: %v", err)
		}
		logging.CPrint(logging.WARN, "got error response", logging.LogFormat{
			"code":    st.Code,
			"message": st.Message,
			"details": st.Details,
		})
		return fmt.Errorf(st.Message)
	} else {
		u := jsonpb.Unmarshaler{AllowUnknownFields: true}
		return u.Unmarshal(resp.Body, response.(proto.Message))
	}
}

// ClientCall selects a client type and execute calling
func ClientCall(path string, method Method, request, response interface{}) error {
	initClient()
	if err := client.Call(context.Background(), path, method, request, response); err != nil {
		logging.VPrint(logging.ERROR, "fail on client call", logging.LogFormat{"err": err})
		return err
	} else {
		printJSON(response)
	}
	return nil
}

func ClientCallWithoutPrintResponse(path string, method Method, request, response interface{}) error {
	initClient()
	err := client.Call(context.Background(), path, method, request, response)
	if err != nil {
		logging.VPrint(logging.ERROR, "fail on client call", logging.LogFormat{"err": err})
	}
	return err
}

func printJSON(data interface{}) {
	m := jsonpb.Marshaler{EmitDefaults: false, Indent: "  "}

	str, err := m.MarshalToString(data.(proto.Message))
	if err != nil {
		logging.VPrint(logging.FATAL, "fail to marshal json", logging.LogFormat{"err": err, "data_type": reflect.TypeOf(data)})
	}

	jww.FEEDBACK.Println(str)
}

var EmptyLogFormat = logging.LogFormat{}

func parseCommandVar(arg string) (key, value string, err error) {
	kv := strings.Split(arg, "=")
	if len(kv) != 2 {
		return "", "", ErrInvalidArgument
	}
	key = strings.TrimSpace(kv[0])
	value = strings.TrimSpace(kv[1])
	if len(key) == 0 || len(value) == 0 {
		return "", "", ErrInvalidArgument
	}
	return strings.ToLower(key), strings.ToLower(value), nil
}

func errorUnknownCommandParam(name string) error {
	return fmt.Errorf("unknown command param: %s", name)
}
