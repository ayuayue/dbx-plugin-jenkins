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

// Jenkins 的 sendError 页面把原因写在 <h1>Error</h1><p> 里，且 <p> 没有闭合标签。
// 这类 400 是可读的业务拒绝，必须把原因透出来，而不是换成通用提示。
func TestAPIErrorExtractsSendErrorPage(t *testing.T) {
	page := `<!DOCTYPE html><html><head><title>Jenkins - Jenkins</title></head><body>
<!-- start of main content ⇒
--><h1>Error</h1><p>` + "\u2018*\u2019" + ` is an unsafe character<!--
⇐ end of main content
--></body></html>`
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html;charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(page))
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	_, err := client.CreateJob(context.Background(), "", "demo", JobKindFreeStyle, "", "")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if apiErr.Kind != ErrRejected {
		t.Errorf("kind = %q, want rejected", apiErr.Kind)
	}
	if !strings.Contains(apiErr.Message, "is an unsafe character") {
		t.Errorf("message = %q, want the Jenkins explanation", apiErr.Message)
	}
	if strings.Contains(apiErr.Message, "<h1") || strings.Contains(apiErr.Message, "main content") {
		t.Errorf("message leaked HTML: %q", apiErr.Message)
	}
}

func TestAPIErrorExtractsDuplicateName(t *testing.T) {
	page := `<!DOCTYPE html><html><body><h1>Error</h1><p>A job already exists with the name ` +
		"\u2018demo\u2019" + `<!-- end --></body></html>`
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(page))
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	_, err := client.CreateJob(context.Background(), "", "demo", JobKindFreeStyle, "", "")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if apiErr.Kind != ErrRejected {
		t.Errorf("kind = %q, want rejected", apiErr.Kind)
	}
	if !strings.Contains(apiErr.Message, "already exists with the name") {
		t.Errorf("message = %q, want the duplicate-name hint", apiErr.Message)
	}
}

// 未闭合的 <p> 必须按注释/标签边界截断，不能把整页正文都当原因。
func TestAPIErrorStopsAtParagraphBoundary(t *testing.T) {
	page := `<html><body><h1>Error</h1><p>No name is specified<!-- end -->` +
		`<div class="footer">Do not include this footer</div></body></html>`
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(page))
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	_, err := client.CreateJob(context.Background(), "", "demo", JobKindFreeStyle, "", "")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if apiErr.Message != "No name is specified" {
		t.Errorf("message = %q, want the exact reason", apiErr.Message)
	}
}

func TestAPIErrorUnescapesHTMLEntities(t *testing.T) {
	page := `<html><body><h1>Error</h1><p>` + "\u2018&lt;\u2019" + ` is an unsafe character<!-- e --></body></html>`
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(page))
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	_, err := client.CreateJob(context.Background(), "", "demo", JobKindFreeStyle, "", "")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if !strings.Contains(apiErr.Message, "\u2018<\u2019 is an unsafe character") {
		t.Errorf("message = %q, want HTML entities decoded", apiErr.Message)
	}
}

// 500 的 Oops 页没有可读原因，但带 Logging ID，运维靠它查服务端日志。
func TestAPIErrorKeepsLoggingID(t *testing.T) {
	page := `<html><body><h1 style="text-align: center"><span style="font-size:50px"> Oops!</span></h1>` +
		`<div id="error-description"><h2 style="text-align: center">A problem occurred while processing the request</h2>` +
		`<p style="text-align: center">Logging ID=44a27356-d701-4205-91ca-ee512fff33c2</div></body></html>`
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(page))
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	_, err := client.CreateJob(context.Background(), "", "demo", JobKindFreeStyle, "", "")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if apiErr.Kind != ErrServer {
		t.Errorf("kind = %q, want server_error", apiErr.Kind)
	}
	if !strings.Contains(apiErr.Message, "Logging ID=44a27356-d701-4205-91ca-ee512fff33c2") {
		t.Errorf("message = %q, want the Jenkins logging ID", apiErr.Message)
	}
}

func TestValidateNameRejectsUnsafeCharacters(t *testing.T) {
	unsafe := []string{"*", ":", "<", ">", "|", "&", ";", "!", "@", "$", "^", "[", "]", `"`}
	for _, char := range unsafe {
		name := "a" + char + "b"
		err := ValidateName(name)
		if err == nil {
			t.Errorf("ValidateName(%q) = nil, want an error", name)
			continue
		}
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Kind != ErrInvalidConfig {
			t.Errorf("ValidateName(%q) kind = %q, want invalid_config", name, apiErr.Kind)
		}
	}
}

func TestValidateNameAcceptsSafeCharacters(t *testing.T) {
	safe := []string{"(", ")", "{", "}", "=", "+", ",", "'", "~", "`", "-", "。", " "}
	for _, char := range safe {
		name := "a" + char + "b"
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) error = %v, want nil", name, err)
		}
	}
	for _, name := range []string{"发布任务", "release-1.0", "my job"} {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) error = %v, want nil", name, err)
		}
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
