package jenkins

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestAPIErrorSummarizesHTMLPage(t *testing.T) {
	page := "<!DOCTYPE html><html><head><title>Jenkins</title></head><body>Stack trace</body></html>"
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html;charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(page))
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	err := client.SaveJobConfig(context.Background(), "demo", "<project/>")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if apiErr.Kind != ErrServer {
		t.Errorf("kind = %q, want server_error", apiErr.Kind)
	}
	if strings.Contains(apiErr.Message, "<html") || strings.Contains(apiErr.Message, "Stack trace") {
		t.Errorf("message leaked HTML: %q", apiErr.Message)
	}
	if !strings.Contains(apiErr.Message, "HTML error page") {
		t.Errorf("message = %q, want an HTML hint", apiErr.Message)
	}
}

func TestAPIErrorKeepsPlainTextBody(t *testing.T) {
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "A job already exists with the name 'demo'", http.StatusBadRequest)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	err := client.SaveJobConfig(context.Background(), "demo", "<project/>")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if !strings.Contains(apiErr.Message, "already exists") {
		t.Errorf("message = %q, want the Jenkins explanation", apiErr.Message)
	}
}
