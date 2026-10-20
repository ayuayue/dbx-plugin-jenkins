package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func splitServerURL(t *testing.T, raw string) (string, string, int) {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", raw, err)
	}
	host, portText, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("SplitHostPort(%q) error = %v", parsed.Host, err)
	}
	var port int
	if _, err := fmt.Sscanf(portText, "%d", &port); err != nil {
		t.Fatalf("invalid port %q: %v", portText, err)
	}
	return parsed.Scheme, host, port
}

func lifecyclePayload(
	t *testing.T,
	serverURL string,
	token string,
	shape string,
	basePath string,
	readOnly bool,
) json.RawMessage {
	t.Helper()
	scheme, host, port := splitServerURL(t, serverURL)

	config := map[string]any{
		"scheme":    scheme,
		"base_path": basePath,
	}
	if readOnly {
		config["read_only"] = true
	}

	connection := map[string]any{
		"id":              "conn-1",
		"name":            "test",
		"host":            host,
		"port":            port,
		"username":        "alice",
		"external_config": config,
	}
	switch shape {
	case "secrets":
		connection["secrets"] = map[string]any{"api_token": token}
	case "connection_secrets":
		connection["connection_secrets"] = map[string]any{"api_token": token}
	case "password":
		connection["password"] = token
	case "top_secrets":
		// 只放到顶层 secrets，验证兜底解析。
	default:
		connection["api_token"] = token
	}

	payload := map[string]any{
		"connection":   connection,
		"runtime":      map[string]any{"host": host, "port": port},
		"connectionId": "conn-1",
	}
	if shape == "top_secrets" {
		payload["secrets"] = map[string]any{"api_token": token}
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return encoded
}

func fakeJenkins(t *testing.T, token string, basePath string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(basePath+"/whoAmI/api/json", func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			user, pass, ok := r.BasicAuth()
			if !ok || user != "alice" || pass != token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		w.Header().Set("X-Jenkins", "2.440.3")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"name":"alice","authenticated":true,"authorities":["authenticated"]}`)
	})
	return httptest.NewServer(mux)
}

type connectionTestResult struct {
	Success  bool   `json:"success"`
	ReadOnly bool   `json:"readOnly"`
	Message  string `json:"message"`
	User     struct {
		Name          string `json:"name"`
		Authenticated bool   `json:"authenticated"`
		Version       string `json:"version"`
	} `json:"user"`
}

func decodeResult[T any](t *testing.T, value any) T {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var decoded T
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return decoded
}

func TestConnectionTestTokenShapes(t *testing.T) {
	const token = "secret-token"
	server := fakeJenkins(t, token, "")
	defer server.Close()

	for _, shape := range []string{"api_token", "secrets", "connection_secrets", "password", "top_secrets"} {
		t.Run(shape, func(t *testing.T) {
			service := NewService()
			result, rpcErr := service.Handle(
				context.Background(),
				"connection/test",
				lifecyclePayload(t, server.URL, token, shape, "", false),
			)
			if rpcErr != nil {
				t.Fatalf("connection/test error = %+v", rpcErr)
			}
			decoded := decodeResult[connectionTestResult](t, result)
			if !decoded.Success || decoded.User.Name != "alice" {
				t.Errorf("result = %+v, want success with alice", decoded)
			}
			if decoded.User.Version != "2.440.3" {
				t.Errorf("version = %q, want 2.440.3", decoded.User.Version)
			}
		})
	}
}

func TestConnectionTestWithBasePath(t *testing.T) {
	server := fakeJenkins(t, "token", "/jenkins")
	defer server.Close()

	service := NewService()
	result, rpcErr := service.Handle(
		context.Background(),
		"connection/test",
		lifecyclePayload(t, server.URL, "token", "api_token", "/jenkins/", false),
	)
	if rpcErr != nil {
		t.Fatalf("connection/test error = %+v", rpcErr)
	}
	decoded := decodeResult[connectionTestResult](t, result)
	if !decoded.Success {
		t.Errorf("result = %+v, want success", decoded)
	}
}

func TestConnectionTestUnauthorized(t *testing.T) {
	server := fakeJenkins(t, "correct-token", "")
	defer server.Close()

	service := NewService()
	_, rpcErr := service.Handle(
		context.Background(),
		"connection/test",
		lifecyclePayload(t, server.URL, "wrong-token", "api_token", "", false),
	)
	if rpcErr == nil {
		t.Fatal("connection/test error = nil, want unauthorized")
	}
	if rpcErr.Kind != "unauthorized" {
		t.Errorf("Kind = %q, want unauthorized", rpcErr.Kind)
	}
}

func TestConnectionTestInvalidConfig(t *testing.T) {
	service := NewService()
	payload := json.RawMessage(`{"connection":{"id":"x","external_config":{"scheme":"https"}},"runtime":{}}`)
	_, rpcErr := service.Handle(context.Background(), "connection/test", payload)
	if rpcErr == nil {
		t.Fatal("connection/test error = nil, want invalid config")
	}
	if rpcErr.Kind != "invalid_config" || rpcErr.Code != -32602 {
		t.Errorf("error = %+v, want kind invalid_config code -32602", rpcErr)
	}
}

func TestConnectDisconnectIdempotent(t *testing.T) {
	server := fakeJenkins(t, "token", "")
	defer server.Close()

	service := NewService()
	payload := lifecyclePayload(t, server.URL, "token", "api_token", "", false)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if _, rpcErr := service.Handle(ctx, "connection/connect", payload); rpcErr != nil {
			t.Fatalf("connect #%d error = %+v", i+1, rpcErr)
		}
	}
	if _, rpcErr := service.Handle(ctx, "jenkins/whoami", json.RawMessage(`{"connectionId":"conn-1"}`)); rpcErr != nil {
		t.Fatalf("whoami after connect error = %+v", rpcErr)
	}

	for i := 0; i < 2; i++ {
		if _, rpcErr := service.Handle(ctx, "connection/disconnect", payload); rpcErr != nil {
			t.Fatalf("disconnect #%d error = %+v", i+1, rpcErr)
		}
	}
	_, rpcErr := service.Handle(ctx, "jenkins/whoami", json.RawMessage(`{"connectionId":"conn-1"}`))
	if rpcErr == nil || rpcErr.Kind != "not_connected" {
		t.Fatalf("whoami after disconnect error = %+v, want not_connected", rpcErr)
	}
}

func TestConnectionTestDoesNotKeepSession(t *testing.T) {
	server := fakeJenkins(t, "token", "")
	defer server.Close()

	service := NewService()
	payload := lifecyclePayload(t, server.URL, "token", "api_token", "", false)
	if _, rpcErr := service.Handle(context.Background(), "connection/test", payload); rpcErr != nil {
		t.Fatalf("connection/test error = %+v", rpcErr)
	}
	_, rpcErr := service.Handle(context.Background(), "jenkins/whoami", json.RawMessage(`{"connectionId":"conn-1"}`))
	if rpcErr == nil || rpcErr.Kind != "not_connected" {
		t.Fatalf("whoami after test error = %+v, want not_connected", rpcErr)
	}
}

func TestReadOnlyFlagFromExternalConfig(t *testing.T) {
	server := fakeJenkins(t, "token", "")
	defer server.Close()

	service := NewService()
	result, rpcErr := service.Handle(
		context.Background(),
		"connection/test",
		lifecyclePayload(t, server.URL, "token", "api_token", "", true),
	)
	if rpcErr != nil {
		t.Fatalf("connection/test error = %+v", rpcErr)
	}
	decoded := decodeResult[connectionTestResult](t, result)
	if !decoded.ReadOnly {
		t.Errorf("ReadOnly = false, want true")
	}
}

func TestConnectRequiresConnectionID(t *testing.T) {
	service := NewService()
	_, rpcErr := service.Handle(context.Background(), "connection/connect", json.RawMessage(`{"connection":{},"runtime":{}}`))
	if rpcErr == nil || rpcErr.Code != -32602 {
		t.Fatalf("error = %+v, want invalid params", rpcErr)
	}
}

func TestUnknownMethodNeedsSession(t *testing.T) {
	service := NewService()
	_, rpcErr := service.Handle(context.Background(), "jenkins/unknown", json.RawMessage(`{"connectionId":"missing"}`))
	if rpcErr == nil || rpcErr.Kind != "not_connected" {
		t.Fatalf("error = %+v, want not_connected", rpcErr)
	}
}
