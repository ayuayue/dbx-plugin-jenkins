package jenkins

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestTriggerBuildWithoutParams(t *testing.T) {
	var path string
	var method string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/crumbIssuer/api/json" {
			http.NotFound(w, r)
			return
		}
		path = r.URL.Path
		method = r.Method
		w.Header().Set("Location", serverURL(r)+"/queue/item/42/")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	queueID, err := client.TriggerBuild(context.Background(), "team/demo", nil)
	if err != nil {
		t.Fatalf("TriggerBuild() error = %v", err)
	}
	if queueID != 42 {
		t.Errorf("queueID = %d, want 42", queueID)
	}
	if path != "/job/team/job/demo/build" || method != http.MethodPost {
		t.Errorf("request = %s %s, want POST /job/team/job/demo/build", method, path)
	}
}

func TestTriggerBuildWithParams(t *testing.T) {
	var path string
	var form url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/crumbIssuer/api/json" {
			http.NotFound(w, r)
			return
		}
		path = r.URL.Path
		_ = r.ParseForm()
		form = r.PostForm
		w.Header().Set("Location", "/queue/item/7/")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	queueID, err := client.TriggerBuild(context.Background(), "demo", url.Values{"BRANCH": {"main"}})
	if err != nil {
		t.Fatalf("TriggerBuild() error = %v", err)
	}
	if queueID != 7 {
		t.Errorf("queueID = %d, want 7", queueID)
	}
	if path != "/job/demo/buildWithParameters" {
		t.Errorf("path = %q, want /job/demo/buildWithParameters", path)
	}
	if form.Get("BRANCH") != "main" {
		t.Errorf("form = %+v, want BRANCH=main", form)
	}
}

func TestTriggerBuildReadOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	_, err := client.TriggerBuild(context.Background(), "demo", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrReadOnly {
		t.Fatalf("error = %v, want read_only", err)
	}
}

func TestStopBuild(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/crumbIssuer/api/json" {
			http.NotFound(w, r)
			return
		}
		path = r.URL.Path
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if err := client.StopBuild(context.Background(), "team/demo", 12); err != nil {
		t.Fatalf("StopBuild() error = %v", err)
	}
	if path != "/job/team/job/demo/12/stop" {
		t.Errorf("path = %q, want /job/team/job/demo/12/stop", path)
	}
}

func TestParseQueueID(t *testing.T) {
	tests := []struct {
		location string
		want     int
	}{
		{location: "http://ci/queue/item/42/", want: 42},
		{location: "/queue/item/43", want: 43},
		{location: "", want: 0},
		{location: "/queue/item/abc/", want: 0},
		{location: "http://ci/job/demo/", want: 0},
	}
	for _, test := range tests {
		t.Run(test.location, func(t *testing.T) {
			if got := parseQueueID(test.location); got != test.want {
				t.Errorf("parseQueueID(%q) = %d, want %d", test.location, got, test.want)
			}
		})
	}
}

func TestTriggerBuildErrorMapsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/crumbIssuer/api/json" {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	_, err := client.TriggerBuild(context.Background(), "demo", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrForbidden {
		t.Fatalf("error = %v, want forbidden", err)
	}
}
