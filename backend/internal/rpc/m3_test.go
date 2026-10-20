package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeJenkinsM3(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/whoAmI/api/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Jenkins", "2.440.3")
		fmt.Fprint(w, `{"name":"alice","authenticated":true}`)
	})
	mux.HandleFunc("/crumbIssuer/api/json", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/json", func(w http.ResponseWriter, r *http.Request) {
		tree := r.URL.Query().Get("tree")
		if strings.Contains(tree, "jobs[") {
			fmt.Fprint(w, `{"jobs":[{"_class":"hudson.model.FreeStyleProject","name":"demo","fullName":"demo","color":"blue"}]}`)
			return
		}
		fmt.Fprint(w, `{"mode":"NORMAL","nodeDescription":"master","numExecutors":4,"quietingDown":false}`)
	})
	mux.HandleFunc("/queue/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"items":[
			{"id":42,"task":{"name":"team/demo","url":"http://ci/job/team/job/demo/"},"why":"Waiting for next available executor","stuck":false,"blocked":false,"buildableIn":1200,"inQueueSince":1700000000000,"actions":[{"causes":[{"shortDescription":"Started by user Alice"}]}]},
			{"id":43,"task":{"name":"build","url":"http://ci/job/build/"},"why":"Blocked by upstream project","stuck":true,"blocked":true,"buildableIn":0,"inQueueSince":1700000001000,"actions":[]}
		]}`)
	})
	mux.HandleFunc("/computer/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"computer":[
			{"displayName":"Built-In Node","offline":false,"temporarilyOffline":false,"numExecutors":4,"idle":false,"assignedLabels":[{"name":"built-in"}],"offlineCauseReason":"","executors":[{"idle":false},{"idle":false},{"idle":true},{"idle":true}]},
			{"displayName":"agent-1","offline":true,"temporarilyOffline":true,"numExecutors":2,"idle":true,"assignedLabels":[{"name":"linux"}],"offlineCauseReason":"maintenance","executors":[{"idle":true},{"idle":true}]}
		]}`)
	})
	mux.HandleFunc("/queue/cancelItem", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Query().Get("id") != "42" {
			t.Errorf("id = %q, want 42", r.URL.Query().Get("id"))
		}
		w.WriteHeader(http.StatusOK)
	})

	return httptest.NewServer(mux)
}

func TestRPCOverview(t *testing.T) {
	server := fakeJenkinsM3(t)
	defer server.Close()
	service := connectM2(t, server, true)

	result, rpcErr := service.Handle(context.Background(), "jenkins/overview", json.RawMessage(`{"connectionId":"conn-1"}`))
	if rpcErr != nil {
		t.Fatalf("overview error = %+v", rpcErr)
	}
	overview := decodeResult[struct {
		Version        string `json:"version"`
		Mode           string `json:"mode"`
		QueueLength    int    `json:"queueLength"`
		NodesOnline    int    `json:"nodesOnline"`
		NodesTotal     int    `json:"nodesTotal"`
		ExecutorsBusy  int    `json:"executorsBusy"`
		ExecutorsTotal int    `json:"executorsTotal"`
	}](t, result)
	if overview.Version != "2.440.3" || overview.Mode != "NORMAL" {
		t.Errorf("overview = %+v, want 2.440.3/NORMAL", overview)
	}
	if overview.QueueLength != 2 || overview.NodesTotal != 2 || overview.NodesOnline != 1 {
		t.Errorf("counts = %+v, want queue 2 nodes 1/2", overview)
	}
	if overview.ExecutorsBusy != 2 || overview.ExecutorsTotal != 6 {
		t.Errorf("executors = %d/%d, want 2/6", overview.ExecutorsBusy, overview.ExecutorsTotal)
	}
}

func TestRPCQueueAndCancel(t *testing.T) {
	server := fakeJenkinsM3(t)
	defer server.Close()
	service := connectM2(t, server, false)
	ctx := context.Background()

	result, rpcErr := service.Handle(ctx, "jenkins/queue", json.RawMessage(`{"connectionId":"conn-1"}`))
	if rpcErr != nil {
		t.Fatalf("queue error = %+v", rpcErr)
	}
	queue := decodeResult[struct {
		Items []struct {
			ID      int    `json:"id"`
			JobName string `json:"jobName"`
			Why     string `json:"why"`
			Cause   string `json:"cause"`
			Stuck   bool   `json:"stuck"`
		} `json:"items"`
	}](t, result)
	if len(queue.Items) != 2 || queue.Items[0].JobName != "team/demo" {
		t.Errorf("queue = %+v, want 2 items with team/demo", queue.Items)
	}
	if queue.Items[0].Cause != "Started by user Alice" || !queue.Items[1].Stuck {
		t.Errorf("queue details = %+v, want Alice cause and stuck item", queue.Items)
	}

	if _, rpcErr := service.Handle(ctx, "jenkins/cancelQueue", json.RawMessage(`{"connectionId":"conn-1","id":42}`)); rpcErr != nil {
		t.Fatalf("cancelQueue error = %+v", rpcErr)
	}
}

func TestRPCQueueReadOnly(t *testing.T) {
	server := fakeJenkinsM3(t)
	defer server.Close()
	service := connectM2(t, server, true)

	_, rpcErr := service.Handle(context.Background(), "jenkins/cancelQueue", json.RawMessage(`{"connectionId":"conn-1","id":42}`))
	if rpcErr == nil || rpcErr.Kind != "read_only" {
		t.Fatalf("error = %+v, want read_only", rpcErr)
	}
}

func TestRPCNodes(t *testing.T) {
	server := fakeJenkinsM3(t)
	defer server.Close()
	service := connectM2(t, server, true)

	result, rpcErr := service.Handle(context.Background(), "jenkins/nodes", json.RawMessage(`{"connectionId":"conn-1"}`))
	if rpcErr != nil {
		t.Fatalf("nodes error = %+v", rpcErr)
	}
	nodes := decodeResult[struct {
		Nodes []struct {
			Name          string   `json:"name"`
			Offline       bool     `json:"offline"`
			Busy          int      `json:"busy"`
			NumExecutors  int      `json:"numExecutors"`
			Labels        []string `json:"labels"`
			OfflineReason string   `json:"offlineReason"`
		} `json:"nodes"`
	}](t, result)
	if len(nodes.Nodes) != 2 {
		t.Fatalf("nodes = %d, want 2", len(nodes.Nodes))
	}
	if nodes.Nodes[0].Busy != 2 || nodes.Nodes[0].Labels[0] != "built-in" {
		t.Errorf("node 0 = %+v, want busy 2 built-in", nodes.Nodes[0])
	}
	if !nodes.Nodes[1].Offline || nodes.Nodes[1].OfflineReason != "maintenance" {
		t.Errorf("node 1 = %+v, want offline maintenance", nodes.Nodes[1])
	}
}
