package jenkins

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetQueue(t *testing.T) {
	var path, tree string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		tree = r.URL.Query().Get("tree")
		fmt.Fprint(w, `{"items":[
			{"id":42,"task":{"name":"team/demo","url":"http://ci/job/team/job/demo/"},"why":"Waiting for next available executor","stuck":false,"blocked":false,"buildableIn":1200,"inQueueSince":1700000000000,"actions":[{"causes":[{"shortDescription":"Started by user Alice"}]}]},
			{"id":43,"task":{"name":"build","url":"http://ci/job/build/"},"why":"Blocked by upstream project","stuck":true,"blocked":true,"buildableIn":0,"inQueueSince":1700000001000,"actions":[]}
		]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	items, err := client.GetQueue(context.Background())
	if err != nil {
		t.Fatalf("GetQueue() error = %v", err)
	}
	if path != "/queue/api/json" {
		t.Errorf("path = %q, want /queue/api/json", path)
	}
	if !strings.Contains(tree, "inQueueSince") {
		t.Errorf("tree = %q, want inQueueSince", tree)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].JobName != "team/demo" || items[0].Cause != "Started by user Alice" {
		t.Errorf("item 42 = %+v, want team/demo with Alice", items[0])
	}
	if !items[1].Stuck || !items[1].Blocked || items[1].Why == "" {
		t.Errorf("item 43 = %+v, want stuck/blocked with reason", items[1])
	}
}

func TestCancelQueueItem(t *testing.T) {
	var path, query, method string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/crumbIssuer/api/json" {
			http.NotFound(w, r)
			return
		}
		path = r.URL.Path
		query = r.URL.Query().Get("id")
		method = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if err := client.CancelQueueItem(context.Background(), 42); err != nil {
		t.Fatalf("CancelQueueItem() error = %v", err)
	}
	if path != "/queue/cancelItem" || query != "42" || method != http.MethodPost {
		t.Errorf("request = %s %s?id=%s, want POST /queue/cancelItem?id=42", method, path, query)
	}
}

func TestCancelQueueItemReadOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	err := client.CancelQueueItem(context.Background(), 42)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrReadOnly {
		t.Fatalf("error = %v, want read_only", err)
	}
}

func TestGetNodes(t *testing.T) {
	var path, tree string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		tree = r.URL.Query().Get("tree")
		fmt.Fprint(w, `{"computer":[
			{"displayName":"Built-In Node","offline":false,"temporarilyOffline":false,"numExecutors":4,"idle":false,"assignedLabels":[{"name":"built-in"}],"offlineCauseReason":"","executors":[{"idle":false},{"idle":false},{"idle":true},{"idle":true}]},
			{"displayName":"agent-1","offline":true,"temporarilyOffline":true,"numExecutors":2,"idle":true,"assignedLabels":[{"name":"linux"}],"offlineCauseReason":"maintenance","executors":[{"idle":true},{"idle":true}]}
		]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	nodes, err := client.GetNodes(context.Background())
	if err != nil {
		t.Fatalf("GetNodes() error = %v", err)
	}
	if path != "/computer/api/json" {
		t.Errorf("path = %q, want /computer/api/json", path)
	}
	if !strings.Contains(tree, "offlineCauseReason") {
		t.Errorf("tree = %q, want offlineCauseReason", tree)
	}
	if !strings.Contains(tree, "executors[idle]") {
		t.Errorf("tree = %q, want executors[idle]", tree)
	}
	if len(nodes) != 2 {
		t.Fatalf("nodes = %d, want 2", len(nodes))
	}
	if nodes[0].Busy != 2 || nodes[0].Labels[0] != "built-in" {
		t.Errorf("node 0 = %+v, want busy 2 with built-in label", nodes[0])
	}
	if !nodes[1].Offline || nodes[1].OfflineReason != "maintenance" {
		t.Errorf("node 1 = %+v, want offline with maintenance", nodes[1])
	}
}

func TestGetNodesFallbackIdleFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 老版本 Jenkins 没有 executors 明细，只有布尔 idle。
		fmt.Fprint(w, `{"computer":[{"displayName":"old-node","offline":false,"numExecutors":2,"idle":true}]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	nodes, err := client.GetNodes(context.Background())
	if err != nil {
		t.Fatalf("GetNodes() error = %v", err)
	}
	if len(nodes) != 1 || nodes[0].Idle != 1 || nodes[0].Busy != 1 {
		t.Fatalf("nodes = %+v, want idle 1 / busy 1", nodes)
	}
}

func TestGetExecutors(t *testing.T) {
	var path, tree string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		tree = r.URL.Query().Get("tree")
		fmt.Fprintf(w, "%s", `{"computer":[
			{"displayName":"","offline":false,"numExecutors":2,"executors":[
				{"idle":false,"currentExecutable":{"fullDisplayName":"团队 » demo-job #12","number":12,"url":"http://ci/jenkins/job/%E5%9B%A2%E9%98%9F/job/demo-job/12/","timestamp":1700000000000,"estimatedDuration":45000,"duration":18000,"building":true,"result":null}},
				{"idle":true,"currentExecutable":null}
			]},
			{"displayName":"agent-1","offline":true,"numExecutors":1,"executors":[
				{"idle":false,"currentExecutable":{"fullDisplayName":"team » release #7","number":7,"url":"/job/team/job/release/7/","timestamp":1700000001000,"estimatedDuration":120000,"duration":60000,"building":true,"result":null}}
			]}
		]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	rows, err := client.GetExecutors(context.Background())
	if err != nil {
		t.Fatalf("GetExecutors() error = %v", err)
	}
	if path != "/computer/api/json" {
		t.Errorf("path = %q, want /computer/api/json", path)
	}
	if !strings.Contains(tree, "currentExecutable[fullDisplayName") {
		t.Errorf("tree = %q, want currentExecutable[fullDisplayName", tree)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3 (空闲执行器也要占位)", len(rows))
	}

	first := rows[0]
	if first.Node != "" || first.Index != 1 || first.Idle || first.Pending {
		t.Errorf("row 0 = %+v, want 内置节点 #1 构建中", first)
	}
	// fullDisplayName 用的是「父 » 子」的展示名，不能当 fullName 用，必须从 url 反解。
	if first.JobName != "团队/demo-job" || first.BuildNumber != 12 || !first.Building {
		t.Errorf("row 0 = %+v, want 团队/demo-job #12 building", first)
	}
	if first.Duration != 18000 || first.EstimatedDuration != 45000 {
		t.Errorf("row 0 timing = %+v, want 18000/45000", first)
	}

	second := rows[1]
	if !second.Idle || second.Index != 2 || second.JobName != "" || second.BuildNumber != 0 {
		t.Errorf("row 1 = %+v, want 空闲 #2", second)
	}

	third := rows[2]
	if third.Node != "agent-1" || !third.NodeOffline {
		t.Errorf("row 2 = %+v, want offline agent-1", third)
	}
	if third.JobName != "team/release" || third.BuildNumber != 7 {
		t.Errorf("row 2 = %+v, want team/release #7", third)
	}
}

func TestGetExecutorsPendingExecutable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 执行器已占位、构建对象还没建好：只有 url，没有 number / fullDisplayName。
		fmt.Fprint(w, `{"computer":[{"displayName":"","offline":false,"numExecutors":1,"executors":[
			{"idle":false,"currentExecutable":{"url":"http://ci/queue/item/42/"}}
		]}]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	rows, err := client.GetExecutors(context.Background())
	if err != nil {
		t.Fatalf("GetExecutors() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if !rows[0].Pending || rows[0].Idle || rows[0].JobName != "" || rows[0].BuildNumber != 0 {
		t.Fatalf("row = %+v, want pending placeholder", rows[0])
	}
}

func TestJobFullNameFromURL(t *testing.T) {
	cases := []struct {
		name     string
		url      string
		expected string
	}{
		{"顶层任务（绝对地址）", "http://ci/job/demo-job/12/", "demo-job"},
		{"带 context path", "http://ci/jenkins/job/demo-job/12/", "demo-job"},
		{"文件夹任务", "http://ci/job/Builds/job/anjun-web/12/", "Builds/anjun-web"},
		{"三层文件夹", "http://ci/job/a/job/b/job/c/9/", "a/b/c"},
		{"相对地址", "/job/team/job/release/7/", "team/release"},
		{"URL 编码的中文", "http://ci/job/%E5%9B%A2%E9%98%9F/job/demo/12/", "团队/demo"},
		{"缺尾斜杠", "http://ci/job/demo/12", "demo"},
		{"带查询串", "http://ci/job/demo/12/?foo=bar", "demo"},
		{"队列项占位没有 job 段", "http://ci/queue/item/42/", ""},
		{"空串", "", ""},
	}
	for _, c := range cases {
		if got := jobFullNameFromURL(c.url); got != c.expected {
			t.Errorf("%s: jobFullNameFromURL(%q) = %q, want %q", c.name, c.url, got, c.expected)
		}
	}
}

func TestJobNameFromDisplayName(t *testing.T) {
	cases := []struct {
		display  string
		number   int
		expected string
	}{
		{"demo-job #12", 12, "demo-job"},
		{"team/release #7", 7, "team/release"},
		{"任务 带空格 #3", 3, "任务 带空格"},
		{"demo-job", 0, "demo-job"},
		{"", 0, ""},
	}
	for _, c := range cases {
		if got := jobNameFromDisplayName(c.display, c.number); got != c.expected {
			t.Errorf("jobNameFromDisplayName(%q, %d) = %q, want %q", c.display, c.number, got, c.expected)
		}
	}
}

func TestGetOverview(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/whoAmI/api/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Jenkins", "2.440.3")
		fmt.Fprint(w, `{"name":"alice","authenticated":true}`)
	})
	mux.HandleFunc("/api/json", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tree") != "mode,nodeDescription,numExecutors,quietingDown" {
			t.Errorf("unexpected tree = %q", r.URL.Query().Get("tree"))
		}
		fmt.Fprint(w, `{"mode":"NORMAL","nodeDescription":"master","numExecutors":4,"quietingDown":true}`)
	})
	mux.HandleFunc("/queue/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"items":[{"id":1,"task":{"name":"a"}},{"id":2,"task":{"name":"b"}}]}`)
	})
	mux.HandleFunc("/computer/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"computer":[
			{"displayName":"master","offline":false,"numExecutors":4,"executors":[{"idle":false},{"idle":false},{"idle":true},{"idle":true}]},
			{"displayName":"agent","offline":true,"numExecutors":2,"executors":[{"idle":false},{"idle":true}]}
		]}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	overview, err := client.GetOverview(context.Background())
	if err != nil {
		t.Fatalf("GetOverview() error = %v", err)
	}
	if overview.Version != "2.440.3" || overview.User != "alice" {
		t.Errorf("identity = %+v, want alice/2.440.3", overview)
	}
	if overview.Mode != "NORMAL" || !overview.QuietingDown {
		t.Errorf("mode = %+v, want NORMAL quiet", overview)
	}
	if overview.QueueLength != 2 {
		t.Errorf("QueueLength = %d, want 2", overview.QueueLength)
	}
	if overview.NodesTotal != 2 || overview.NodesOnline != 1 {
		t.Errorf("nodes = %d/%d, want 1/2", overview.NodesOnline, overview.NodesTotal)
	}
	if overview.ExecutorsBusy != 3 || overview.ExecutorsTotal != 6 {
		t.Errorf("executors = %d/%d, want 3/6", overview.ExecutorsBusy, overview.ExecutorsTotal)
	}
}
