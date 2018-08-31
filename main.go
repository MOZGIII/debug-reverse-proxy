package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	targetURL string
	addr      string
)

func run() error {
	addr = os.Getenv("ADDR")
	if addr == "" {
		addr = ":9000"
	}

	targetURL = os.Getenv("TARGET_URL")
	if targetURL == "" {
		return fmt.Errorf("Empty TARGET_URL")
	}

	u, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	log("Starting reverse proxy at %q for %q\n", addr, targetURL)
	rp := newSingleHostReverseProxy(u)
	return http.ListenAndServe(addr, rp)
}

func main() {
	if err := run(); err != nil {
		log("Fatal error: %s\n", err)
	}
}

func log(format string, a ...interface{}) {
	if _, err := fmt.Fprintf(os.Stderr, format, a...); err != nil {
		panic(err)
	}
}

// Adopted from net/http/httputil/reverseproxy.go
func newSingleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery

	director := func(req *http.Request) {
		dump, err := httputil.DumpRequest(req, true)
		if err != nil {
			log("Error constructing the request dump: %s\n", err)
		}
		if err := writeDump(dump, eventTypeRequest); err != nil {
			log("Error writing the request dump: %s\n", err)
		}

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	modifyResponse := func(res *http.Response) error {
		dump, err := httputil.DumpResponse(res, true)
		if err != nil {
			log("Error constructing the response dump: %s\n", err)
		}
		if err := writeDump(dump, eventTypeResponse); err != nil {
			log("Error writing the response dump: %s\n", err)
		}
		return nil
	}

	return &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: modifyResponse,
	}
}

// Adopted from net/http/httputil/reverseproxy.go
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

type eventType int

const (
	eventTypeRequest eventType = iota
	eventTypeResponse
)

func (e eventType) String() string {
	switch e {
	case eventTypeRequest:
		return "request"
	case eventTypeResponse:
		return "response"
	default:
		return fmt.Sprintf("unknown (%d)", int(e))
	}
}

type jsonMessage struct {
	Msg      string    `json:"msg"`
	DumpKind string    `json:"dump_kind"`
	Dump     string    `json:"dump"`
	Time     time.Time `json:"time"`
}

func writeDumpPlainText(data []byte, _ eventType) error {
	_, err := os.Stdout.Write(data)
	return err
}

func writeDumpJSON(data []byte, t eventType) error {
	m := jsonMessage{
		Msg:      t.String(),
		DumpKind: t.String(),
		Dump:     string(data),
		Time:     time.Now().UTC(),
	}

	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(&m)
}

var writeDump = writeDumpPlainText

func init() {
	if os.Getenv("DUMP_JSON") == "true" {
		writeDump = writeDumpJSON
	}
}
