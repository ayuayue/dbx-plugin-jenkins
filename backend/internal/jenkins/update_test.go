package jenkins

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestParseJobFields(t *testing.T) {
	freeFields, err := ParseJobFields(jobConfigFixture, "team/demo")
	if err != nil {
		t.Fatalf("ParseJobFields(free) error = %v", err)
	}
	if freeFields.Kind != string(JobKindFreeStyle) {
		t.Errorf("kind = %q, want freestyle", freeFields.Kind)
	}
	if freeFields.Title != "demo" {
		t.Errorf("title = %q, want demo", freeFields.Title)
	}
	if freeFields.Description != "old description" {
		t.Errorf("description = %q", freeFields.Description)
	}
	if freeFields.Disabled {
		t.Error("disabled = true, want false")
	}
	if freeFields.Script != "" {
		t.Errorf("script = %q, want empty", freeFields.Script)
	}

	pipelineFields, err := ParseJobFields(pipelineConfigFixture, "pipe")
	if err != nil {
		t.Fatalf("ParseJobFields(pipeline) error = %v", err)
	}
	if pipelineFields.Kind != string(JobKindPipeline) {
		t.Errorf("kind = %q, want pipeline", pipelineFields.Kind)
	}
	if pipelineFields.Script != "echo old" {
		t.Errorf("script = %q, want echo old", pipelineFields.Script)
	}
}

func TestParseListViewFields(t *testing.T) {
	cfg, err := ParseListViewFields(listViewConfigFixture)
	if err != nil {
		t.Fatalf("ParseListViewFields() error = %v", err)
	}
	if cfg.Name != "Builds" {
		t.Errorf("name = %q", cfg.Name)
	}
	if cfg.Description != "old description" {
		t.Errorf("description = %q", cfg.Description)
	}
	if len(cfg.JobNames) != 1 || cfg.JobNames[0] != "demo" {
		t.Errorf("jobNames = %+v, want [demo]", cfg.JobNames)
	}
	if cfg.IncludeRegex != "" {
		t.Errorf("includeRegex = %q, want empty", cfg.IncludeRegex)
	}
	if cfg.Recurse {
		t.Error("recurse = true, want false")
	}
}

func TestUpdateJobDescriptionPreservesConfig(t *testing.T) {
	var (
		posted     string
		saveCalls  atomic.Int32
		getCalls   atomic.Int32
		renameHits atomic.Int32
	)
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.EscapedPath() == "/job/demo/config.xml" && r.Method == http.MethodGet:
			getCalls.Add(1)
			fmt.Fprint(w, jobConfigFixture)
		case r.URL.EscapedPath() == "/job/demo/config.xml" && r.Method == http.MethodPost:
			saveCalls.Add(1)
			body, _ := io.ReadAll(r.Body)
			posted = string(body)
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "doRename"):
			renameHits.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	fullName, err := client.UpdateJob(context.Background(), "demo", JobUpdate{
		Description:    "updated",
		SetDescription: true,
	})
	if err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}
	if fullName != "demo" {
		t.Errorf("fullName = %q, want demo", fullName)
	}
	if getCalls.Load() != 1 || saveCalls.Load() != 1 {
		t.Errorf("get/save calls = %d/%d, want 1/1", getCalls.Load(), saveCalls.Load())
	}
	if renameHits.Load() != 0 {
		t.Errorf("rename calls = %d, want 0", renameHits.Load())
	}
	if !strings.Contains(posted, "<description>updated</description>") {
		t.Errorf("description not updated: %s", posted)
	}
	if !strings.Contains(posted, "hudson.plugins.git.GitSCM") {
		t.Errorf("SCM dropped: %s", posted)
	}
}

func TestUpdateJobRenamesBeforeSaving(t *testing.T) {
	var (
		renamedTo string
		savedPath string
	)
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.EscapedPath() == "/job/demo/doRename":
			renamedTo = r.URL.Query().Get("newName")
			w.WriteHeader(http.StatusOK)
		case r.URL.EscapedPath() == "/job/demo-v2/config.xml" && r.Method == http.MethodGet:
			fmt.Fprint(w, jobConfigFixture)
		case r.URL.EscapedPath() == "/job/demo-v2/config.xml" && r.Method == http.MethodPost:
			savedPath = r.URL.EscapedPath()
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	fullName, err := client.UpdateJob(context.Background(), "demo", JobUpdate{
		Title:        "demo-v2",
		SetTitle:     true,
		Disabled:     true,
		SetDisabled:  true,
	})
	if err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}
	if fullName != "demo-v2" {
		t.Errorf("fullName = %q, want demo-v2", fullName)
	}
	if renamedTo != "demo-v2" {
		t.Errorf("renamedTo = %q", renamedTo)
	}
	if savedPath != "/job/demo-v2/config.xml" {
		t.Errorf("saved path = %q", savedPath)
	}
}

func TestUpdateJobPipelineScript(t *testing.T) {
	var posted string
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/job/pipe/config.xml" {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodGet {
			fmt.Fprint(w, pipelineConfigFixture)
			return
		}
		body, _ := io.ReadAll(r.Body)
		posted = string(body)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if _, err := client.UpdateJob(context.Background(), "pipe", JobUpdate{
		Script:    "echo updated",
		SetScript: true,
	}); err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}
	if !strings.Contains(posted, "<script>echo updated</script>") {
		t.Errorf("script not updated: %s", posted)
	}
	if !strings.Contains(posted, "<sandbox>true</sandbox>") {
		t.Errorf("sandbox dropped: %s", posted)
	}
}

func TestUpdateJobWithoutFieldsSkipsRequests(t *testing.T) {
	var calls atomic.Int32
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	fullName, err := client.UpdateJob(context.Background(), "demo", JobUpdate{})
	if err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}
	if fullName != "demo" {
		t.Errorf("fullName = %q", fullName)
	}
	if calls.Load() != 0 {
		t.Errorf("calls = %d, want 0", calls.Load())
	}
}

func TestUpdateJobRejectsReadOnly(t *testing.T) {
	var calls atomic.Int32
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	err := func() error {
		_, err := client.UpdateJob(context.Background(), "demo", JobUpdate{
			Description:    "x",
			SetDescription: true,
		})
		return err
	}()
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrReadOnly {
		t.Fatalf("error = %v, want read_only", err)
	}
	if calls.Load() != 0 {
		t.Errorf("calls = %d, want 0", calls.Load())
	}
}

func TestUpdateViewKeepsUnchangedFields(t *testing.T) {
	var posted string
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/view/Builds/config.xml" {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodGet {
			fmt.Fprint(w, listViewConfigFixture)
			return
		}
		body, _ := io.ReadAll(r.Body)
		posted = string(body)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if err := client.UpdateView(context.Background(), "Builds", ViewUpdate{
		Description:    "new description",
		SetDescription: true,
	}); err != nil {
		t.Fatalf("UpdateView() error = %v", err)
	}
	if !strings.Contains(posted, "<description>new description</description>") {
		t.Errorf("description not updated: %s", posted)
	}
	if !strings.Contains(posted, "<string>demo</string>") {
		t.Errorf("job names were dropped: %s", posted)
	}
	if !strings.Contains(posted, "<hudson.views.StatusColumn/>") {
		t.Errorf("columns dropped: %s", posted)
	}
}

func TestUpdateViewReplacesJobs(t *testing.T) {
	var posted string
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/view/Builds/config.xml" {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodGet {
			fmt.Fprint(w, listViewConfigFixture)
			return
		}
		body, _ := io.ReadAll(r.Body)
		posted = string(body)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if err := client.UpdateView(context.Background(), "Builds", ViewUpdate{
		JobNames:    []string{"api", "web"},
		SetJobNames: true,
	}); err != nil {
		t.Fatalf("UpdateView() error = %v", err)
	}
	if !strings.Contains(posted, "<string>api</string>") ||
		!strings.Contains(posted, "<string>web</string>") {
		t.Errorf("job names not replaced: %s", posted)
	}
	if strings.Contains(posted, "<string>demo</string>") {
		t.Errorf("old job names still present: %s", posted)
	}
	if !strings.Contains(posted, "<description>old description</description>") {
		t.Errorf("description was lost: %s", posted)
	}
}

func TestUpdateViewRejectsReadOnly(t *testing.T) {
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	err := client.UpdateView(context.Background(), "Builds", ViewUpdate{
		Description:    "x",
		SetDescription: true,
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrReadOnly {
		t.Fatalf("error = %v, want read_only", err)
	}
}
