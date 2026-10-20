package jenkins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "plain", input: "demo"},
		{name: "chinese", input: "发布任务"},
		{name: "space", input: "my job"},
		{name: "version dots", input: "release-1.0"},
		{name: "underscore", input: "build_main"},
		{name: "max length", input: strings.Repeat("a", 255)},
		{name: "empty", input: "", wantErr: true},
		{name: "blank", input: "   ", wantErr: true},
		{name: "slash", input: "a/b", wantErr: true},
		{name: "backslash", input: `a\b`, wantErr: true},
		{name: "question", input: "a?b", wantErr: true},
		{name: "hash", input: "a#b", wantErr: true},
		{name: "percent", input: "a%b", wantErr: true},
		{name: "dot dot", input: "..", wantErr: true},
		{name: "dot dot inside", input: "a..b", wantErr: true},
		{name: "control char", input: "a\x00b", wantErr: true},
		{name: "too long", input: strings.Repeat("a", 256), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateName(test.input)
			if test.wantErr {
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.Kind != ErrInvalidConfig {
					t.Fatalf("ValidateName(%q) error = %v, want invalid_config", test.input, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateName(%q) error = %v, want nil", test.input, err)
			}
		})
	}
}

func TestJobConfigPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "single", input: "demo", want: "/job/demo/config.xml"},
		{name: "nested", input: "team/api", want: "/job/team/job/api/config.xml"},
		{name: "spaces", input: "Release 1.0", want: "/job/Release%201.0/config.xml"},
		{name: "empty", input: "", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := JobConfigPath(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("JobConfigPath(%q) error = nil, want an error", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("JobConfigPath(%q) error = %v", test.input, err)
			}
			if got != test.want {
				t.Errorf("JobConfigPath(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestViewConfigPath(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{input: "Builds", want: "/view/Builds/config.xml"},
		{input: "Release 1.0", want: "/view/Release%201.0/config.xml"},
		{input: "", wantErr: true},
	}

	for _, test := range tests {
		got, err := ViewConfigPath(test.input)
		if test.wantErr {
			if err == nil {
				t.Fatalf("ViewConfigPath(%q) error = nil, want an error", test.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ViewConfigPath(%q) error = %v", test.input, err)
		}
		if got != test.want {
			t.Errorf("ViewConfigPath(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestGetJobConfig(t *testing.T) {
	var (
		gotPath   string
		gotAccept string
	)
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/job/team/job/api/config.xml" {
			http.NotFound(w, r)
			return
		}
		gotPath = r.URL.EscapedPath()
		gotAccept = r.Header.Get("Accept")
		fmt.Fprint(w, "<project><description>hi</description></project>")
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	xmlText, err := client.GetJobConfig(context.Background(), "team/api")
	if err != nil {
		t.Fatalf("GetJobConfig() error = %v", err)
	}
	if xmlText != "<project><description>hi</description></project>" {
		t.Errorf("xml = %q", xmlText)
	}
	if gotPath != "/job/team/job/api/config.xml" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAccept != "application/xml" {
		t.Errorf("Accept = %q, want application/xml", gotAccept)
	}
}

func TestSaveJobConfigUsesXMLAndCrumb(t *testing.T) {
	var (
		gotMethod      string
		gotPath        string
		gotContentType string
		gotCrumb       string
		gotBody        string
	)
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/job/demo/config.xml" {
			http.NotFound(w, r)
			return
		}
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotContentType = r.Header.Get("Content-Type")
		gotCrumb = r.Header.Get("Jenkins-Crumb")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	const xmlText = "<project><description>saved</description></project>"
	if err := client.SaveJobConfig(context.Background(), "demo", xmlText); err != nil {
		t.Fatalf("SaveJobConfig() error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/job/demo/config.xml" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.HasPrefix(gotContentType, "text/xml") {
		t.Errorf("Content-Type = %q, want text/xml", gotContentType)
	}
	if gotCrumb != "crumb-1" {
		t.Errorf("crumb = %q, want crumb-1", gotCrumb)
	}
	if gotBody != xmlText {
		t.Errorf("body = %q, want %q", gotBody, xmlText)
	}
}

func TestPostTreatsRedirectAsSuccess(t *testing.T) {
	var calls atomic.Int32
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/job/demo/doDelete" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Location", serverURL(r)+"/job/demo/")
		w.WriteHeader(http.StatusFound)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if err := client.DeleteJob(context.Background(), "demo"); err != nil {
		t.Fatalf("DeleteJob() error = %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1 (must not follow the redirect)", calls.Load())
	}
}

func TestCreateJobFreeStyle(t *testing.T) {
	var (
		gotPath string
		gotName string
		gotBody string
	)
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/createItem" {
			http.NotFound(w, r)
			return
		}
		gotPath = r.URL.Path
		gotName = r.URL.Query().Get("name")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	fullName, err := client.CreateJob(
		context.Background(), "", "demo",
		JobKindFreeStyle, "hello & world", "",
	)
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if fullName != "demo" {
		t.Errorf("fullName = %q, want demo", fullName)
	}
	if gotPath != "/createItem" || gotName != "demo" {
		t.Errorf("path = %q name = %q", gotPath, gotName)
	}
	if !strings.Contains(gotBody, "<project>") {
		t.Errorf("body missing <project>: %s", gotBody)
	}
	if !strings.Contains(gotBody, "<description>hello &amp; world</description>") {
		t.Errorf("description not escaped: %s", gotBody)
	}
}

func TestCreateJobInFolder(t *testing.T) {
	var gotPath, gotName string
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/job/team/createItem" {
			http.NotFound(w, r)
			return
		}
		gotPath = r.URL.Path
		gotName = r.URL.Query().Get("name")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	fullName, err := client.CreateJob(context.Background(), "team", "demo", JobKindFreeStyle, "", "")
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if fullName != "team/demo" {
		t.Errorf("fullName = %q, want team/demo", fullName)
	}
	if gotPath != "/job/team/createItem" || gotName != "demo" {
		t.Errorf("path = %q name = %q", gotPath, gotName)
	}
}

func TestCreateJobPipeline(t *testing.T) {
	var gotBody string
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/createItem" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if _, err := client.CreateJob(
		context.Background(), "", "pipe",
		JobKindPipeline, "nightly", "echo <hi>",
	); err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if !strings.Contains(gotBody, "<flow-definition>") {
		t.Errorf("body missing flow-definition: %s", gotBody)
	}
	if !strings.Contains(gotBody, "CpsFlowDefinition") {
		t.Errorf("body missing CpsFlowDefinition: %s", gotBody)
	}
	if !strings.Contains(gotBody, "<script>echo &lt;hi&gt;</script>") {
		t.Errorf("script not escaped: %s", gotBody)
	}
}

func TestCreateJobRejectsUnknownKind(t *testing.T) {
	var calls atomic.Int32
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	_, err := client.CreateJob(context.Background(), "", "demo", JobKind("folder"), "", "")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrInvalidConfig {
		t.Fatalf("error = %v, want invalid_config", err)
	}
	if calls.Load() != 0 {
		t.Errorf("calls = %d, want 0", calls.Load())
	}
}

func TestRenameJob(t *testing.T) {
	var gotPath, gotNewName string
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/job/team/job/api/doRename" {
			http.NotFound(w, r)
			return
		}
		gotPath = r.URL.EscapedPath()
		gotNewName = r.URL.Query().Get("newName")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	fullName, err := client.RenameJob(context.Background(), "team/api", "api-v2")
	if err != nil {
		t.Fatalf("RenameJob() error = %v", err)
	}
	if fullName != "team/api-v2" {
		t.Errorf("fullName = %q, want team/api-v2", fullName)
	}
	if gotPath != "/job/team/job/api/doRename" || gotNewName != "api-v2" {
		t.Errorf("path = %q newName = %q", gotPath, gotNewName)
	}
}

func TestDeleteJob(t *testing.T) {
	var gotPath string
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/job/team/job/api/doDelete" {
			http.NotFound(w, r)
			return
		}
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if err := client.DeleteJob(context.Background(), "team/api"); err != nil {
		t.Fatalf("DeleteJob() error = %v", err)
	}
	if gotPath != "/job/team/job/api/doDelete" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestCreateView(t *testing.T) {
	var form url.Values
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/createView" {
			http.NotFound(w, r)
			return
		}
		_ = r.ParseForm()
		form = r.PostForm
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	err := client.CreateView(context.Background(), ListViewConfig{
		Name:         "CRUD-View",
		Description:  "managed by dbx",
		JobNames:     []string{"api", "web"},
		IncludeRegex: "^release-.*",
		Recurse:      true,
	})
	if err != nil {
		t.Fatalf("CreateView() error = %v", err)
	}

	if form.Get("name") != "CRUD-View" {
		t.Errorf("form name = %q", form.Get("name"))
	}
	if form.Get("mode") != "hudson.model.ListView" {
		t.Errorf("form mode = %q", form.Get("mode"))
	}

	var payload struct {
		Name         string           `json:"name"`
		Mode         string           `json:"mode"`
		Description  string           `json:"description"`
		JobNames     []string         `json:"jobNames"`
		IncludeRegex string           `json:"includeRegex"`
		Recurse      bool             `json:"recurse"`
		Columns      []map[string]any `json:"columns"`
	}
	raw := form.Get("json")
	if raw == "" {
		t.Fatal("form json is empty")
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, raw = %s", err, raw)
	}
	if payload.Name != "CRUD-View" || payload.Mode != "hudson.model.ListView" {
		t.Errorf("json name/mode = %q/%q", payload.Name, payload.Mode)
	}
	if payload.Description != "managed by dbx" {
		t.Errorf("json description = %q", payload.Description)
	}
	if len(payload.JobNames) != 2 || payload.JobNames[0] != "api" || payload.JobNames[1] != "web" {
		t.Errorf("json jobNames = %+v, want [api web]", payload.JobNames)
	}
	if payload.IncludeRegex != "^release-.*" {
		t.Errorf("json includeRegex = %q", payload.IncludeRegex)
	}
	if !payload.Recurse {
		t.Error("json recurse = false, want true")
	}
	if len(payload.Columns) != 7 {
		t.Errorf("json columns = %d, want 7", len(payload.Columns))
	}
}

func TestCreateViewRejectsInvalidName(t *testing.T) {
	var calls atomic.Int32
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	err := client.CreateView(context.Background(), ListViewConfig{Name: "bad/name"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrInvalidConfig {
		t.Fatalf("error = %v, want invalid_config", err)
	}
	if calls.Load() != 0 {
		t.Errorf("calls = %d, want 0", calls.Load())
	}
}

func TestViewConfigReadWrite(t *testing.T) {
	var (
		gotGetPath  string
		gotPostPath string
		gotBody     string
	)
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/view/Builds/config.xml" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			gotGetPath = r.URL.EscapedPath()
			fmt.Fprint(w, "<hudson.model.ListView><name>Builds</name></hudson.model.ListView>")
		case http.MethodPost:
			gotPostPath = r.URL.EscapedPath()
			body, _ := io.ReadAll(r.Body)
			gotBody = string(body)
			w.WriteHeader(http.StatusOK)
		}
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	xmlText, err := client.GetViewConfig(context.Background(), "Builds")
	if err != nil {
		t.Fatalf("GetViewConfig() error = %v", err)
	}
	if !strings.Contains(xmlText, "<name>Builds</name>") {
		t.Errorf("xml = %q", xmlText)
	}
	if err := client.SaveViewConfig(context.Background(), "Builds", xmlText); err != nil {
		t.Fatalf("SaveViewConfig() error = %v", err)
	}
	if gotGetPath != "/view/Builds/config.xml" || gotPostPath != "/view/Builds/config.xml" {
		t.Errorf("paths = %q / %q", gotGetPath, gotPostPath)
	}
	if gotBody != xmlText {
		t.Errorf("body = %q, want %q", gotBody, xmlText)
	}
}

func TestDeleteView(t *testing.T) {
	var gotPath string
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/view/Ops/doDelete" {
			http.NotFound(w, r)
			return
		}
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if err := client.DeleteView(context.Background(), "Ops"); err != nil {
		t.Fatalf("DeleteView() error = %v", err)
	}
	if gotPath != "/view/Ops/doDelete" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestConfigWritesRespectReadOnly(t *testing.T) {
	var calls atomic.Int32
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	ctx := context.Background()

	writes := []struct {
		name string
		call func() error
	}{
		{name: "SaveJobConfig", call: func() error {
			return client.SaveJobConfig(ctx, "demo", "<project/>")
		}},
		{name: "CreateJob", call: func() error {
			_, err := client.CreateJob(ctx, "", "demo", JobKindFreeStyle, "", "")
			return err
		}},
		{name: "RenameJob", call: func() error {
			_, err := client.RenameJob(ctx, "demo", "demo2")
			return err
		}},
		{name: "DeleteJob", call: func() error {
			return client.DeleteJob(ctx, "demo")
		}},
		{name: "SaveViewConfig", call: func() error {
			return client.SaveViewConfig(ctx, "Builds", "<hudson.model.ListView/>")
		}},
		{name: "CreateView", call: func() error {
			return client.CreateView(ctx, ListViewConfig{Name: "New"})
		}},
		{name: "DeleteView", call: func() error {
			return client.DeleteView(ctx, "Builds")
		}},
	}

	for _, write := range writes {
		t.Run(write.name, func(t *testing.T) {
			err := write.call()
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.Kind != ErrReadOnly {
				t.Fatalf("error = %v, want read_only", err)
			}
		})
	}

	if calls.Load() != 0 {
		t.Errorf("server calls = %d, want 0", calls.Load())
	}
}

func TestSaveJobConfigSurfacesPermissionHint(t *testing.T) {
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(
			w,
			"user is missing the Item/Create permission",
			http.StatusForbidden,
		)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	err := client.SaveJobConfig(context.Background(), "demo", "<project/>")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if apiErr.Kind != ErrForbidden {
		t.Errorf("kind = %q, want forbidden", apiErr.Kind)
	}
	if !strings.Contains(apiErr.Message, "Item/Create") {
		t.Errorf("message = %q, want permission hint preserved", apiErr.Message)
	}
}

// newConfigServer 构造带 crumb 的最小 Jenkins 假服务；业务路径由 handle 处理。
func newConfigServer(t *testing.T, handle http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/crumbIssuer/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"crumb":"crumb-1","crumbRequestField":"Jenkins-Crumb"}`)
	})
	mux.HandleFunc("/", handle)
	return httptest.NewServer(mux)
}
