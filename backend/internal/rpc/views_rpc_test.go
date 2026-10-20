package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func fakeJenkinsViews(t *testing.T) (*httptest.Server, *atomic.Value) {
	t.Helper()
	var lastPath atomic.Value
	lastPath.Store("")

	mux := http.NewServeMux()
	mux.HandleFunc("/whoAmI/api/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Jenkins", "2.586")
		fmt.Fprint(w, `{"name":"alice","authenticated":true}`)
	})
	mux.HandleFunc("/crumbIssuer/api/json", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/json", func(w http.ResponseWriter, r *http.Request) {
		lastPath.Store(r.URL.Path)
		tree := r.URL.Query().Get("tree")
		if strings.HasPrefix(tree, "views") {
			fmt.Fprint(w, `{"views":[
				{"name":"All","description":"All jobs","url":"http://ci/"},
				{"name":"Builds","description":"Build jobs","url":"http://ci/view/Builds/"},
				{"name":"Ops","description":"Operations","url":"http://ci/view/Ops/"}
			]}`)
			return
		}
		fmt.Fprint(w, `{"jobs":[
			{"_class":"hudson.model.FreeStyleProject","name":"demo","fullName":"demo","color":"blue"},
			{"_class":"hudson.model.FreeStyleProject","name":"param-demo","fullName":"param-demo","color":"red"},
			{"_class":"hudson.model.FreeStyleProject","name":"slow-job","fullName":"slow-job","color":"blue"}
		]}`)
	})
	mux.HandleFunc("/view/Builds/api/json", func(w http.ResponseWriter, r *http.Request) {
		lastPath.Store(r.URL.Path)
		fmt.Fprint(w, `{"jobs":[
			{"_class":"hudson.model.FreeStyleProject","name":"demo","fullName":"demo","color":"blue"},
			{"_class":"hudson.model.FreeStyleProject","name":"param-demo","fullName":"param-demo","color":"red"}
		]}`)
	})
	mux.HandleFunc("/view/Ops/api/json", func(w http.ResponseWriter, r *http.Request) {
		lastPath.Store(r.URL.Path)
		fmt.Fprint(w, `{"jobs":[{"_class":"hudson.model.FreeStyleProject","name":"slow-job","fullName":"slow-job","color":"blue"}]}`)
	})
	mux.HandleFunc("/job/team/api/json", func(w http.ResponseWriter, r *http.Request) {
		lastPath.Store(r.URL.Path)
		fmt.Fprint(w, `{"jobs":[{"_class":"hudson.model.FreeStyleProject","name":"inside","fullName":"team/inside","color":"blue"}]}`)
	})

	return httptest.NewServer(mux), &lastPath
}

func TestRPCViews(t *testing.T) {
	server, _ := fakeJenkinsViews(t)
	defer server.Close()
	service := connectM2(t, server, true)

	result := handleMCP(t, service, "jenkins/views", map[string]any{"connectionId": "conn-1"})
	views := decodeResult[struct {
		Views []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"views"`
	}](t, result)
	if len(views.Views) != 3 || views.Views[1].Name != "Builds" {
		t.Fatalf("views = %+v, want All/Builds/Ops", views.Views)
	}
	if views.Views[1].Description != "Build jobs" {
		t.Errorf("description = %q, want Build jobs", views.Views[1].Description)
	}
}

func TestRPCJobsByView(t *testing.T) {
	server, _ := fakeJenkinsViews(t)
	defer server.Close()
	service := connectM2(t, server, true)
	ctx := context.Background()

	result, rpcErr := service.Handle(ctx, "jenkins/jobs", json.RawMessage(`{"connectionId":"conn-1","view":"Builds"}`))
	if rpcErr != nil {
		t.Fatalf("Builds error = %+v", rpcErr)
	}
	builds := decodeResult[struct {
		Jobs []struct {
			Name string `json:"name"`
		} `json:"jobs"`
	}](t, result)
	if len(builds.Jobs) != 2 {
		t.Fatalf("Builds jobs = %d, want 2", len(builds.Jobs))
	}
	for _, job := range builds.Jobs {
		if job.Name == "slow-job" {
			t.Errorf("Builds view leaked slow-job")
		}
	}

	result, rpcErr = service.Handle(ctx, "jenkins/jobs", json.RawMessage(`{"connectionId":"conn-1","view":"Ops"}`))
	if rpcErr != nil {
		t.Fatalf("Ops error = %+v", rpcErr)
	}
	ops := decodeResult[struct {
		Jobs []struct {
			Name string `json:"name"`
		} `json:"jobs"`
	}](t, result)
	if len(ops.Jobs) != 1 || ops.Jobs[0].Name != "slow-job" {
		t.Errorf("Ops jobs = %+v, want slow-job only", ops.Jobs)
	}
}

func TestRPCJobsFolderWinsOverView(t *testing.T) {
	server, lastPath := fakeJenkinsViews(t)
	defer server.Close()
	service := connectM2(t, server, true)

	result, rpcErr := service.Handle(
		context.Background(),
		"jenkins/jobs",
		json.RawMessage(`{"connectionId":"conn-1","folder":"team","view":"Builds"}`),
	)
	if rpcErr != nil {
		t.Fatalf("jobs error = %+v", rpcErr)
	}
	jobs := decodeResult[struct {
		Jobs []struct {
			FullName string `json:"fullName"`
		} `json:"jobs"`
	}](t, result)
	if len(jobs.Jobs) != 1 || jobs.Jobs[0].FullName != "team/inside" {
		t.Errorf("jobs = %+v, want team/inside", jobs.Jobs)
	}
	if path := lastPath.Load().(string); path != "/job/team/api/json" {
		t.Errorf("last path = %q, want /job/team/api/json", path)
	}
}

func TestMCPListJobsByView(t *testing.T) {
	server, _ := fakeJenkinsViews(t)
	defer server.Close()
	service := connectM2(t, server, true)

	result := handleMCP(t, service, "mcp/call", map[string]any{
		"connectionId": "conn-1",
		"tool":         "list_jobs",
		"arguments":    map[string]any{"view": "Ops"},
	})
	call := decodeResult[mcpCallResult](t, result)
	if call.IsError {
		t.Fatalf("list_jobs error = %+v", call)
	}
	text := call.Content[0].Text
	if !strings.Contains(text, "slow-job") || strings.Contains(text, "param-demo") {
		t.Errorf("view filter failed: %s", text)
	}
}

func TestMCPListJobsSchemaHasView(t *testing.T) {
	server, _ := fakeJenkinsViews(t)
	defer server.Close()
	service := connectM2(t, server, true)

	result := handleMCP(t, service, "mcp/tools", map[string]any{"connectionId": "conn-1"})
	tools := decodeResult[mcpToolList](t, result)
	for _, tool := range tools.Tools {
		if tool.Name != "list_jobs" {
			continue
		}
		properties, _ := tool.InputSchema["properties"].(map[string]any)
		if _, ok := properties["view"]; !ok {
			t.Errorf("list_jobs schema missing view: %+v", properties)
		}
		return
	}
	t.Fatal("list_jobs tool not found")
}
