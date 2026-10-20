package jenkins

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGetViews(t *testing.T) {
	var path, tree string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		tree = r.URL.Query().Get("tree")
		fmt.Fprint(w, `{"views":[
			{"name":"All","description":"All jobs","url":"http://ci/"},
			{"name":"Builds","description":"Build jobs","url":"http://ci/view/Builds/"},
			{"name":"Ops","description":"Operations","url":"http://ci/view/Ops/"}
		]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	views, err := client.GetViews(context.Background())
	if err != nil {
		t.Fatalf("GetViews() error = %v", err)
	}
	if path != "/api/json" {
		t.Errorf("path = %q, want /api/json", path)
	}
	if tree != "views[name,description,url]" {
		t.Errorf("tree = %q, want views tree", tree)
	}
	if len(views) != 3 || views[1].Name != "Builds" || views[1].Description != "Build jobs" {
		t.Errorf("views = %+v, want All/Builds/Ops", views)
	}
}

func TestGetViewJobs(t *testing.T) {
	var path, tree string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		tree = r.URL.Query().Get("tree")
		fmt.Fprint(w, `{"jobs":[
			{"_class":"hudson.model.FreeStyleProject","name":"demo","color":"red","lastBuild":{"number":7,"result":"FAILURE","building":false}},
			{"_class":"com.cloudbees.hudson.plugins.folder.Folder","name":"team","fullName":"team","color":"blue"},
			{"_class":"org.jenkinsci.plugins.workflow.job.WorkflowJob","name":"api","fullName":"api","color":"blue_anime"}
		]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	jobs, err := client.GetViewJobs(context.Background(), "Builds")
	if err != nil {
		t.Fatalf("GetViewJobs() error = %v", err)
	}
	if path != "/view/Builds/api/json" {
		t.Errorf("path = %q, want /view/Builds/api/json", path)
	}
	if !strings.Contains(tree, "lastBuild[number,result") {
		t.Errorf("tree = %q, want lastBuild fields", tree)
	}
	if len(jobs) != 3 {
		t.Fatalf("jobs = %d, want 3", len(jobs))
	}
	if jobs[0].Name != "team" || !jobs[0].IsFolder {
		t.Errorf("first job = %+v, want folder first", jobs[0])
	}
	byName := map[string]JobSummary{}
	for _, job := range jobs {
		byName[job.Name] = job
	}
	if byName["demo"].Status != StatusFailure || byName["demo"].FullName != "demo" {
		t.Errorf("demo = %+v, want failure with inferred fullName", byName["demo"])
	}
	if byName["api"].Status != StatusBuilding || !byName["api"].Building {
		t.Errorf("api = %+v, want building", byName["api"])
	}
}

func TestGetViewJobsEscapesName(t *testing.T) {
	var escaped string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		escaped = r.URL.EscapedPath()
		fmt.Fprint(w, `{"jobs":[]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	if _, err := client.GetViewJobs(context.Background(), "Release 1.0"); err != nil {
		t.Fatalf("GetViewJobs() error = %v", err)
	}
	if escaped != "/view/Release%201.0/api/json" {
		t.Errorf("escaped path = %q, want /view/Release%%201.0/api/json", escaped)
	}
}

func TestGetViewJobsEmpty(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprint(w, `{"jobs":[]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	_, err := client.GetViewJobs(context.Background(), "  ")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrInvalidConfig {
		t.Fatalf("error = %v, want invalid_config", err)
	}
	if calls.Load() != 0 {
		t.Errorf("server calls = %d, want 0", calls.Load())
	}
}
