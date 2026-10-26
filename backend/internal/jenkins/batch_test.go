package jenkins

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// batchFixture 描述一个任务在假 Jenkins 上的行为。
type batchFixture struct {
	buildable bool
	inQueue   bool
	color     string
	// params 是 parameterDefinitions 的 JSON 数组片段；留空表示无参任务。
	params string
}

// batchTrigger 记录一次批量触发打出的 POST。
type batchTrigger struct {
	job  string
	path string
	form url.Values
}

// batchRecorder 收集假 Jenkins 收到的请求，供断言使用。
type batchRecorder struct {
	mu       sync.Mutex
	triggers []batchTrigger
	gets     []string
	requests int
}

func (r *batchRecorder) addTrigger(item batchTrigger) {
	r.mu.Lock()
	r.triggers = append(r.triggers, item)
	r.mu.Unlock()
}

func (r *batchRecorder) addGet(job string) {
	r.mu.Lock()
	r.gets = append(r.gets, job)
	r.mu.Unlock()
}

func (r *batchRecorder) requestCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.requests
}

func (r *batchRecorder) triggerFor(job string) (batchTrigger, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.triggers {
		if item.job == job {
			return item, true
		}
	}
	return batchTrigger{}, false
}

// jobPathName 把 /job/team/job/demo[...] 反解成 fullName。
func jobPathName(path string) string {
	trimmed := strings.TrimPrefix(path, "/")
	parts := strings.Split(trimmed, "/")
	names := make([]string, 0, len(parts)/2)
	for index := 0; index < len(parts); index += 2 {
		if parts[index] != "job" {
			break
		}
		if index+1 >= len(parts) {
			break
		}
		names = append(names, parts[index+1])
	}
	return strings.Join(names, "/")
}

// newBatchServer 构造批量构建用的假 Jenkins：
// GET /job/<name>/api/json 返回任务可构建信息，POST .../build 返回队列项。
func newBatchServer(t *testing.T, fixtures map[string]batchFixture) (*httptest.Server, *batchRecorder) {
	t.Helper()
	recorder := &batchRecorder{}
	var queueID int64 = 20

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder.mu.Lock()
		recorder.requests++
		recorder.mu.Unlock()

		if r.URL.Path == "/crumbIssuer/api/json" {
			http.NotFound(w, r)
			return
		}

		switch {
		case strings.HasSuffix(r.URL.Path, "/api/json"):
			job := jobPathName(strings.TrimSuffix(r.URL.Path, "/api/json"))
			recorder.addGet(job)
			fixture, ok := fixtures[job]
			if !ok {
				http.NotFound(w, r)
				return
			}
			body := `{"name":"` + job + `","fullName":"` + job + `","buildable":` + strconv.FormatBool(fixture.buildable) +
				`,"inQueue":` + strconv.FormatBool(fixture.inQueue) + `,"color":"` + fixture.color + `"`
			if fixture.params != "" {
				body += `,"property":[{"parameterDefinitions":` + fixture.params + `}]`
			}
			body += `}`
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))

		case strings.HasSuffix(r.URL.Path, "/build"), strings.HasSuffix(r.URL.Path, "/buildWithParameters"):
			path := r.URL.Path
			job := jobPathName(strings.TrimSuffix(strings.TrimSuffix(path, "/buildWithParameters"), "/build"))
			if _, ok := fixtures[job]; !ok {
				http.NotFound(w, r)
				return
			}
			// denied 任务用于验证「单条失败不中断其余」。
			if job == "denied" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			if err := r.ParseForm(); err != nil {
				t.Errorf("ParseForm() error = %v", err)
			}
			recorder.addTrigger(batchTrigger{job: job, path: path, form: r.PostForm})
			id := atomic.AddInt64(&queueID, 1)
			w.Header().Set("Location", "/queue/item/"+strconv.FormatInt(id, 10)+"/")
			w.WriteHeader(http.StatusCreated)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server, recorder
}

func TestBatchBuildUsesDefaultsForParameterizedJob(t *testing.T) {
	server, recorder := newBatchServer(t, map[string]batchFixture{
		"param-demo": {
			buildable: true,
			color:     "blue",
			params: `[
				{"name":"BRANCH","type":"StringParameterDefinition","defaultParameterValue":{"value":"main"}},
				{"name":"DRY_RUN","type":"BooleanParameterDefinition","defaultParameterValue":{"value":true}},
				{"name":"COUNT","type":"StringParameterDefinition","defaultParameterValue":{"value":3}}
			]`,
		},
	})
	client := newTestClient(t, server.URL, false)

	results, err := client.BatchBuild(context.Background(), []string{"param-demo"}, BatchBuildOptions{})
	if err != nil {
		t.Fatalf("BatchBuild() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if !results[0].Success || results[0].Skipped || results[0].QueueID == 0 {
		t.Errorf("result = %+v, want success with queue id", results[0])
	}
	if results[0].Job != "param-demo" {
		t.Errorf("job = %q, want param-demo", results[0].Job)
	}

	trigger, ok := recorder.triggerFor("param-demo")
	if !ok {
		t.Fatal("param-demo was not triggered")
	}
	if trigger.path != "/job/param-demo/buildWithParameters" {
		t.Errorf("path = %q, want /job/param-demo/buildWithParameters", trigger.path)
	}
	if trigger.form.Get("BRANCH") != "main" {
		t.Errorf("BRANCH = %q, want main", trigger.form.Get("BRANCH"))
	}
	if trigger.form.Get("DRY_RUN") != "true" {
		t.Errorf("DRY_RUN = %q, want true", trigger.form.Get("DRY_RUN"))
	}
	if trigger.form.Get("COUNT") != "3" {
		t.Errorf("COUNT = %q, want 3", trigger.form.Get("COUNT"))
	}
}

func TestBatchBuildSkipsNotBuildable(t *testing.T) {
	server, recorder := newBatchServer(t, map[string]batchFixture{
		"disabled-job": {buildable: false, color: "disabled"},
	})
	client := newTestClient(t, server.URL, false)

	results, err := client.BatchBuild(context.Background(), []string{"disabled-job"}, BatchBuildOptions{})
	if err != nil {
		t.Fatalf("BatchBuild() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if !results[0].Skipped || results[0].Reason != "not_buildable" || results[0].Success {
		t.Errorf("result = %+v, want skipped/not_buildable", results[0])
	}
	if _, ok := recorder.triggerFor("disabled-job"); ok {
		t.Error("disabled job must not be triggered")
	}
}

func TestBatchBuildSkipBusy(t *testing.T) {
	tests := []struct {
		name       string
		fixture    batchFixture
		skipBusy   bool
		wantReason string
		wantQueued bool
	}{
		{
			name:       "queued job skipped",
			fixture:    batchFixture{buildable: true, color: "blue", inQueue: true},
			skipBusy:   true,
			wantReason: "in_queue",
			wantQueued: false,
		},
		{
			name:       "building job skipped",
			fixture:    batchFixture{buildable: true, color: "blue_anime"},
			skipBusy:   true,
			wantReason: "building",
			wantQueued: false,
		},
		{
			name:       "queued job triggered when skipBusy off",
			fixture:    batchFixture{buildable: true, color: "blue", inQueue: true},
			skipBusy:   false,
			wantReason: "",
			wantQueued: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, recorder := newBatchServer(t, map[string]batchFixture{"busy-job": test.fixture})
			client := newTestClient(t, server.URL, false)

			results, err := client.BatchBuild(
				context.Background(),
				[]string{"busy-job"},
				BatchBuildOptions{SkipBusy: test.skipBusy},
			)
			if err != nil {
				t.Fatalf("BatchBuild() error = %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("results = %d, want 1", len(results))
			}
			if results[0].Reason != test.wantReason {
				t.Errorf("reason = %q, want %q", results[0].Reason, test.wantReason)
			}
			_, triggered := recorder.triggerFor("busy-job")
			if triggered != test.wantQueued {
				t.Errorf("triggered = %v, want %v", triggered, test.wantQueued)
			}
			if test.wantQueued && (!results[0].Success || results[0].Skipped) {
				t.Errorf("result = %+v, want queued", results[0])
			}
			if !test.wantQueued && (!results[0].Skipped || results[0].Success) {
				t.Errorf("result = %+v, want skipped", results[0])
			}
		})
	}
}

func TestBatchBuildContinuesAfterFailure(t *testing.T) {
	server, recorder := newBatchServer(t, map[string]batchFixture{
		"ok-1":   {buildable: true, color: "blue"},
		"denied": {buildable: true, color: "blue"},
		"ok-2":   {buildable: true, color: "blue"},
	})
	client := newTestClient(t, server.URL, false)

	results, err := client.BatchBuild(
		context.Background(),
		[]string{"ok-1", "denied", "ok-2"},
		BatchBuildOptions{},
	)
	if err != nil {
		t.Fatalf("BatchBuild() error = %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	if results[0].Job != "ok-1" || results[1].Job != "denied" || results[2].Job != "ok-2" {
		t.Errorf("order = %v, want input order", []string{results[0].Job, results[1].Job, results[2].Job})
	}
	if !results[0].Success || !results[2].Success {
		t.Errorf("ok jobs = %+v / %+v, want success", results[0], results[2])
	}
	if results[1].Success || results[1].Skipped {
		t.Errorf("denied = %+v, want failure", results[1])
	}
	if results[1].Reason != "forbidden" || results[1].Error == "" {
		t.Errorf("denied reason = %q error = %q, want forbidden with detail", results[1].Reason, results[1].Error)
	}
	if _, ok := recorder.triggerFor("ok-2"); !ok {
		t.Error("ok-2 must still be triggered after denied failed")
	}
}

func TestBatchBuildReadOnly(t *testing.T) {
	server, recorder := newBatchServer(t, map[string]batchFixture{
		"demo": {buildable: true, color: "blue"},
	})
	client := newTestClient(t, server.URL, true)

	_, err := client.BatchBuild(context.Background(), []string{"demo"}, BatchBuildOptions{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrReadOnly {
		t.Fatalf("error = %v, want read_only", err)
	}
	if count := recorder.requestCount(); count != 0 {
		t.Errorf("requests = %d, want 0 (fail fast before any request)", count)
	}
}

func TestBatchBuildReadConcurrencyCapped(t *testing.T) {
	fixtures := map[string]batchFixture{}
	names := make([]string, 0, 8)
	for index := 0; index < 8; index++ {
		name := "job-" + strconv.Itoa(index)
		fixtures[name] = batchFixture{buildable: true, color: "blue"}
		names = append(names, name)
	}

	var inFlight int32
	var maxInFlight int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/crumbIssuer/api/json" {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/api/json") {
			current := atomic.AddInt32(&inFlight, 1)
			for {
				observed := atomic.LoadInt32(&maxInFlight)
				if current <= observed || atomic.CompareAndSwapInt32(&maxInFlight, observed, current) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			atomic.AddInt32(&inFlight, -1)

			job := jobPathName(strings.TrimSuffix(r.URL.Path, "/api/json"))
			_, _ = w.Write([]byte(`{"name":"` + job + `","fullName":"` + job + `","buildable":true,"inQueue":false,"color":"blue"}`))
			return
		}
		w.Header().Set("Location", "/queue/item/9/")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	results, err := client.BatchBuild(context.Background(), names, BatchBuildOptions{MaxConcurrency: 2})
	if err != nil {
		t.Fatalf("BatchBuild() error = %v", err)
	}
	if len(results) != len(names) {
		t.Fatalf("results = %d, want %d", len(results), len(names))
	}
	if max := atomic.LoadInt32(&maxInFlight); max > 2 {
		t.Errorf("max in-flight reads = %d, want <= 2", max)
	}
}

func TestBatchBuildEmptyList(t *testing.T) {
	client := newTestClient(t, "http://127.0.0.1:1", false)
	results, err := client.BatchBuild(context.Background(), nil, BatchBuildOptions{})
	if err != nil {
		t.Fatalf("BatchBuild() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("results = %d, want 0", len(results))
	}
}

func TestBatchBuildAppliesSharedParams(t *testing.T) {
	server, recorder := newBatchServer(t, map[string]batchFixture{
		"param-demo": {
			buildable: true,
			color:     "blue",
			params: `[
				{"name":"BRANCH","type":"StringParameterDefinition","defaultParameterValue":{"value":"main"}},
				{"name":"COUNT","type":"StringParameterDefinition","defaultParameterValue":{"value":3}}
			]`,
		},
	})
	client := newTestClient(t, server.URL, false)

	results, err := client.BatchBuild(context.Background(), []string{"param-demo"}, BatchBuildOptions{
		Params: url.Values{"BRANCH": {"release/1.2"}, "UNKNOWN": {"x"}},
	})
	if err != nil {
		t.Fatalf("BatchBuild() error = %v", err)
	}
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("results = %+v, want 1 success", results)
	}

	trigger, ok := recorder.triggerFor("param-demo")
	if !ok {
		t.Fatal("param-demo was not triggered")
	}
	if trigger.path != "/job/param-demo/buildWithParameters" {
		t.Errorf("path = %q, want buildWithParameters", trigger.path)
	}
	if value := trigger.form.Get("BRANCH"); value != "release/1.2" {
		t.Errorf("BRANCH = %q, want release/1.2", value)
	}
	if value := trigger.form.Get("COUNT"); value != "3" {
		t.Errorf("COUNT = %q, want default 3", value)
	}
	if _, exists := trigger.form["UNKNOWN"]; exists {
		t.Errorf("UNKNOWN must not be submitted: %+v", trigger.form)
	}
}

func TestBatchBuildSharedParamsIgnoredForPlainJob(t *testing.T) {
	server, recorder := newBatchServer(t, map[string]batchFixture{
		"plain": {buildable: true, color: "blue"},
	})
	client := newTestClient(t, server.URL, false)

	if _, err := client.BatchBuild(context.Background(), []string{"plain"}, BatchBuildOptions{
		Params: url.Values{"BRANCH": {"release/1.2"}},
	}); err != nil {
		t.Fatalf("BatchBuild() error = %v", err)
	}

	trigger, ok := recorder.triggerFor("plain")
	if !ok {
		t.Fatal("plain was not triggered")
	}
	if trigger.path != "/job/plain/build" {
		t.Errorf("path = %q, want /job/plain/build (no params declared)", trigger.path)
	}
	if value := trigger.form.Get("BRANCH"); value != "" {
		t.Errorf("BRANCH = %q, want empty (shared param must be filtered out)", value)
	}
}

func TestInspectBatch(t *testing.T) {
	server, _ := newBatchServer(t, map[string]batchFixture{
		"param-demo": {
			buildable: true,
			color:     "blue",
			params: `[
				{"name":"BRANCH","type":"StringParameterDefinition","description":"git branch","defaultParameterValue":{"value":"main"}},
				{"name":"ENV","type":"ChoiceParameterDefinition","choices":["dev","prod"],"defaultParameterValue":{"value":"dev"}}
			]`,
		},
		"busy-demo": {buildable: true, color: "blue_anime", inQueue: true},
	})
	client := newTestClient(t, server.URL, false)

	targets, err := client.InspectBatch(context.Background(), []string{"param-demo", "busy-demo"}, 0)
	if err != nil {
		t.Fatalf("InspectBatch() error = %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(targets))
	}

	param := targets[0]
	if param.FullName != "param-demo" || param.Name != "param-demo" {
		t.Errorf("first target = %+v, want param-demo", param)
	}
	if param.Status != StatusSuccess || param.Building || !param.Buildable {
		t.Errorf("param-demo state = %+v, want success/buildable/not building", param)
	}
	if len(param.Parameters) != 2 {
		t.Fatalf("parameters = %+v, want 2", param.Parameters)
	}
	if param.Parameters[0].Name != "BRANCH" || param.Parameters[0].DefaultValue != "main" {
		t.Errorf("BRANCH = %+v, want default main", param.Parameters[0])
	}
	if param.Parameters[1].Type != "ChoiceParameterDefinition" ||
		len(param.Parameters[1].Choices) != 2 ||
		param.Parameters[1].Choices[1] != "prod" {
		t.Errorf("ENV = %+v, want choice dev/prod", param.Parameters[1])
	}

	busy := targets[1]
	if busy.Status != StatusBuilding || !busy.Building || !busy.InQueue {
		t.Errorf("busy-demo = %+v, want building and queued", busy)
	}
}
