package handler

import (
	"bytes"
	"errors"
	"io"
	"net/http"
)

const maxMCPResponse = 10 << 20

var errMCPResponseTooLarge = errors.New("response_too_large: 业务响应超过 10 MiB，请使用分页")

// UseBackend installs an in-process REST adapter at startup. Both entry points
// execute the same DTO validation, authorization, handlers and business services.
// It intentionally keeps the REST boundary while service-specific adapters are
// migrated; it never opens a socket or depends on the server's listening address.
func (h *MCPHandler) UseBackend(backend http.Handler, origins []string) {
	h.httpClient.Transport = backendTransport{backend}
	h.baseURL = "http://mcp.internal"
	h.origins = append([]string(nil), origins...)
}

type backendTransport struct{ handler http.Handler }

func (t backendTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		defer req.Body.Close()
	}
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	w := &boundedResponse{header: make(http.Header)}
	t.handler.ServeHTTP(w, req)
	if w.overflow {
		return nil, errMCPResponseTooLarge
	}
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return &http.Response{StatusCode: w.status, Header: w.header, Body: io.NopCloser(bytes.NewReader(w.body.Bytes())), Request: req}, nil
}

type boundedResponse struct {
	header   http.Header
	status   int
	body     bytes.Buffer
	overflow bool
}

func (w *boundedResponse) Header() http.Header { return w.header }
func (w *boundedResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *boundedResponse) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.body.Len()+len(p) > maxMCPResponse {
		w.overflow = true
		return 0, errMCPResponseTooLarge
	}
	return w.body.Write(p)
}
