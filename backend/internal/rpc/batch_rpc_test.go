package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// fakeJenkinsBatch 是批量触发用的假 Jenkins：两类任务（普通 / 带参）+ 一个排队中的任务。
func fakeJenkinsBatch(t *testing.T, queuedTriggers *int32) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/whoAmI/api/json", func(w http.ResponseWriter, r *http.Request) {
		if user, pass, ok := r.BasicAuth(); !ok || user != "alice" || pass != "token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `{"name":"alice","authenticated":true}`)
	})
	mux.HandleFunc("/crumbIssuer/api/json", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	mux.HandleFunc("/job/demo/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"name":"demo","fullName":"demo","buildable":true,"inQueue":false,"color":"blue"}`)
	})
	mux.HandleFunc("/job/param-demo/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"name":"param-demo","fullName":"param-demo","buildable":true,"inQueue":false,"color":"blue",
			"property":[{"parameterDefinitions":[
				{"name":"BRANCH","type":"StringParameterDefinition","defaultParameterValue":{"value":"main"}}
			]}]}`)
	})
	mux.HandleFunc("/job/queued/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"name":"queued","fullName":"queued","buildable":true,"inQueue":true,"color":"blue"}`)
	})

	mux.HandleFunc("/job/demo/build", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/queue/item/101/")
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("/job/param-demo/buildWithParameters", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		if branch := r.PostForm.Get("BRANCH"); branch != "main" {
			t.Errorf("BRANCH = %q, want main", branch)
		}
		w.Header().Set("Location", "/queue/item/102/")
		w.WriteHeader(http.StatusCreated)
	})
	queuedHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(queuedTriggers, 1)
		w.Header().Set("Location", "/queue/item/103/")
		w.WriteHeader(http.StatusCreated)
	}
	mux.HandleFunc("/job/queued/build", queuedHandler)
	mux.HandleFunc("/job/queued/buildWithParameters", queuedHandler)

	return httptest.NewServer(mux)
}

func connectBatchTest(t *testing.T, server *httptest.Server) *Service {
	t.Helper()
	service := NewService()
	payload := lifecyclePayload(t, server.URL, "token", "api_token", "", false)
	if _, rpcErr := service.Handle(context.Background(), "connection/connect", payload); rpcErr != nil {
		t.Fatalf("connect error = %+v", rpcErr)
	}
	return service
}

func TestRPCBatchBuild(t *testing.T) {
	var queuedTriggers int32
	server := fakeJenkinsBatch(t, &queuedTriggers)
	defer server.Close()
	service := connectBatchTest(t, server)

	result, rpcErr := service.Handle(
		context.Background(),
		"jenkins/batchBuild",
		json.RawMessage(`{"connectionId":"conn-1","jobs":["demo","param-demo"]}`),
	)
	if rpcErr != nil {
		t.Fatalf("batchBuild error = %+v", rpcErr)
	}

	decoded := decodeResult[struct {
		Results []struct {
			Job     string `json:"job"`
			Success bool   `json:"success"`
			Skipped bool   `json:"skipped"`
			QueueID int    `json:"queueId"`
			Reason  string `json:"reason"`
		} `json:"results"`
	}](t, result)

	if len(decoded.Results) != 2 {
		t.Fatalf("results = %+v, want 2", decoded.Results)
	}
	if decoded.Results[0].Job != "demo" || !decoded.Results[0].Success || decoded.Results[0].QueueID != 101 {
		t.Errorf("first = %+v, want demo success 101", decoded.Results[0])
	}
	if decoded.Results[1].Job != "param-demo" || !decoded.Results[1].Success || decoded.Results[1].QueueID != 102 {
		t.Errorf("second = %+v, want param-demo success 102", decoded.Results[1])
	}
}

func TestRPCBatchBuildSkipBusyDefault(t *testing.T) {
	var queuedTriggers int32
	server := fakeJenkinsBatch(t, &queuedTriggers)
	defer server.Close()
	service := connectBatchTest(t, server)

	result, rpcErr := service.Handle(
		context.Background(),
		"jenkins/batchBuild",
		json.RawMessage(`{"connectionId":"conn-1","jobs":["queued"]}`),
	)
	if rpcErr != nil {
		t.Fatalf("batchBuild error = %+v", rpcErr)
	}

	decoded := decodeResult[struct {
		Results []struct {
			Success bool   `json:"success"`
			Skipped bool   `json:"skipped"`
			Reason  string `json:"reason"`
		} `json:"results"`
	}](t, result)
	if len(decoded.Results) != 1 {
		t.Fatalf("results = %+v, want 1", decoded.Results)
	}
	if !decoded.Results[0].Skipped || decoded.Results[0].Success || decoded.Results[0].Reason != "in_queue" {
		t.Errorf("result = %+v, want skipped/in_queue by default", decoded.Results[0])
	}
	if count := atomic.LoadInt32(&queuedTriggers); count != 0 {
		t.Errorf("queued triggers = %d, want 0 (skipBusy defaults to true)", count)
	}
}

func TestRPCBatchBuildRequiresJobs(t *testing.T) {
	var queuedTriggers int32
	server := fakeJenkinsBatch(t, &queuedTriggers)
	defer server.Close()
	service := connectBatchTest(t, server)

	for _, payload := range []string{
		`{"connectionId":"conn-1"}`,
		`{"connectionId":"conn-1","jobs":[]}`,
	} {
		_, rpcErr := service.Handle(context.Background(), "jenkins/batchBuild", json.RawMessage(payload))
		if rpcErr == nil || rpcErr.Kind != "invalid_params" {
			t.Errorf("payload %s: error = %+v, want invalid_params", payload, rpcErr)
		}
	}
}

func TestRPCBatchInspect(t *testing.T) {
	var queuedTriggers int32
	server := fakeJenkinsBatch(t, &queuedTriggers)
	defer server.Close()
	service := connectBatchTest(t, server)

	result, rpcErr := service.Handle(
		context.Background(),
		"jenkins/batchInspect",
		json.RawMessage(`{"connectionId":"conn-1","jobs":["demo","param-demo"]}`),
	)
	if rpcErr != nil {
		t.Fatalf("batchInspect error = %+v", rpcErr)
	}

	decoded := decodeResult[struct {
		Targets []struct {
			Name       string `json:"name"`
			FullName   string `json:"fullName"`
			Status     string `json:"status"`
			Buildable  bool   `json:"buildable"`
			Parameters []struct {
				Name         string   `json:"name"`
				Type         string   `json:"type"`
				DefaultValue string   `json:"defaultValue"`
				Choices      []string `json:"choices"`
			} `json:"parameters"`
		} `json:"targets"`
	}](t, result)

	if len(decoded.Targets) != 2 {
		t.Fatalf("targets = %+v, want 2", decoded.Targets)
	}
	if decoded.Targets[0].FullName != "demo" || len(decoded.Targets[0].Parameters) != 0 {
		t.Errorf("demo = %+v, want no parameters", decoded.Targets[0])
	}
	param := decoded.Targets[1]
	if param.FullName != "param-demo" || !param.Buildable || param.Status != "success" {
		t.Errorf("param-demo = %+v, want buildable success", param)
	}
	if len(param.Parameters) != 1 || param.Parameters[0].Name != "BRANCH" ||
		param.Parameters[0].DefaultValue != "main" ||
		param.Parameters[0].Type != "StringParameterDefinition" {
		t.Errorf("param-demo parameters = %+v, want BRANCH=main", param.Parameters)
	}
}

func TestRPCBatchInspectRequiresJobs(t *testing.T) {
	var queuedTriggers int32
	server := fakeJenkinsBatch(t, &queuedTriggers)
	defer server.Close()
	service := connectBatchTest(t, server)

	for _, payload := range []string{
		`{"connectionId":"conn-1"}`,
		`{"connectionId":"conn-1","jobs":[]}`,
	} {
		_, rpcErr := service.Handle(context.Background(), "jenkins/batchInspect", json.RawMessage(payload))
		if rpcErr == nil || rpcErr.Kind != "invalid_params" {
			t.Errorf("payload %s: error = %+v, want invalid_params", payload, rpcErr)
		}
	}
}
