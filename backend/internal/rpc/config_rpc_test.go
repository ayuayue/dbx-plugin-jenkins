package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type configCapture struct {
	mu       sync.Mutex
	requests []string
	bodies   map[string]string
}

func (c *configCapture) record(r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	c.mu.Lock()
	c.requests = append(c.requests, r.Method+" "+r.URL.RequestURI())
	if c.bodies == nil {
		c.bodies = map[string]string{}
	}
	c.bodies[r.Method+" "+r.URL.EscapedPath()] = string(body)
	c.mu.Unlock()
}

func (c *configCapture) contains(fragment string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, request := range c.requests {
		if strings.Contains(request, fragment) {
			return true
		}
	}
	return false
}

func (c *configCapture) body(key string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bodies[key]
}

const rpcJobConfig = `<?xml version='1.1' encoding='UTF-8'?>
<project>
  <description>old</description>
  <scm class="hudson.scm.NullSCM"/>
  <builders/>
  <disabled>false</disabled>
</project>`

const rpcPipelineConfig = `<?xml version='1.1' encoding='UTF-8'?>
<flow-definition>
  <description>old</description>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition">
    <script>echo old</script>
    <sandbox>true</sandbox>
  </definition>
  <disabled>false</disabled>
</flow-definition>`

const rpcListViewConfig = `<?xml version='1.1' encoding='UTF-8'?>
<hudson.model.ListView>
  <name>Builds</name>
  <description>old</description>
  <jobNames>
    <comparator class="hudson.util.CaseInsensitiveComparator"/>
    <string>demo</string>
  </jobNames>
  <recurse>false</recurse>
</hudson.model.ListView>`

func fakeJenkinsConfig(t *testing.T) (*httptest.Server, *configCapture) {
	t.Helper()
	capture := &configCapture{}

	mux := http.NewServeMux()
	mux.HandleFunc("/whoAmI/api/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Jenkins", "2.586")
		fmt.Fprint(w, `{"name":"alice","authenticated":true}`)
	})
	mux.HandleFunc("/crumbIssuer/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"crumb":"crumb-1","crumbRequestField":"Jenkins-Crumb"}`)
	})

	configHandler := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			capture.record(r)
			if r.Method == http.MethodGet {
				fmt.Fprint(w, body)
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	}

	mux.HandleFunc("/job/demo/config.xml", configHandler(rpcJobConfig))
	mux.HandleFunc("/job/pipe/config.xml", configHandler(rpcPipelineConfig))
	mux.HandleFunc("/job/demo/doRename", func(w http.ResponseWriter, r *http.Request) {
		capture.record(r)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/job/demo/doDelete", func(w http.ResponseWriter, r *http.Request) {
		capture.record(r)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/createItem", func(w http.ResponseWriter, r *http.Request) {
		capture.record(r)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/view/Builds/config.xml", configHandler(rpcListViewConfig))
	mux.HandleFunc("/view/Builds/configSubmit", func(w http.ResponseWriter, r *http.Request) {
		capture.record(r)
		w.WriteHeader(http.StatusFound)
	})
	mux.HandleFunc("/view/Builds2/config.xml", configHandler(rpcListViewConfig))
	mux.HandleFunc("/api/json", func(w http.ResponseWriter, r *http.Request) {
		capture.record(r)
		fmt.Fprint(w, `{"views":[{"name":"Builds"},{"name":"Ops"}]}`)
	})
	mux.HandleFunc("/view/Builds/doDelete", func(w http.ResponseWriter, r *http.Request) {
		capture.record(r)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/createView", func(w http.ResponseWriter, r *http.Request) {
		capture.record(r)
		w.WriteHeader(http.StatusOK)
	})

	return httptest.NewServer(mux), capture
}

func TestRPCJobConfig(t *testing.T) {
	server, _ := fakeJenkinsConfig(t)
	defer server.Close()
	service := connectM2(t, server, false)

	result, rpcErr := service.Handle(
		context.Background(),
		"jenkins/jobConfig",
		json.RawMessage(`{"connectionId":"conn-1","job":"demo"}`),
	)
	if rpcErr != nil {
		t.Fatalf("jobConfig error = %+v", rpcErr)
	}
	payload := decodeResult[struct {
		XML    string `json:"xml"`
		Fields struct {
			Title       string `json:"title"`
			Kind        string `json:"kind"`
			Description string `json:"description"`
			Disabled    bool   `json:"disabled"`
		} `json:"fields"`
	}](t, result)
	if !strings.Contains(payload.XML, "<description>old</description>") {
		t.Errorf("xml = %q", payload.XML)
	}
	if payload.Fields.Kind != "freestyle" || payload.Fields.Title != "demo" {
		t.Errorf("fields = %+v", payload.Fields)
	}
	if payload.Fields.Description != "old" {
		t.Errorf("description = %q", payload.Fields.Description)
	}
}

func TestRPCUpdateJob(t *testing.T) {
	server, capture := fakeJenkinsConfig(t)
	defer server.Close()
	service := connectM2(t, server, false)

	result, rpcErr := service.Handle(
		context.Background(),
		"jenkins/updateJob",
		json.RawMessage(`{"connectionId":"conn-1","job":"demo","fields":{"description":"new"}}`),
	)
	if rpcErr != nil {
		t.Fatalf("updateJob error = %+v", rpcErr)
	}
	payload := decodeResult[struct {
		Success  bool   `json:"success"`
		FullName string `json:"fullName"`
	}](t, result)
	if !payload.Success || payload.FullName != "demo" {
		t.Errorf("result = %+v", payload)
	}
	body := capture.body("POST /job/demo/config.xml")
	if !strings.Contains(body, "<description>new</description>") {
		t.Errorf("posted body = %q", body)
	}
	if !strings.Contains(body, "<scm class=\"hudson.scm.NullSCM\"/>") {
		t.Errorf("SCM dropped: %q", body)
	}
}

func TestRPCUpdateJobPipelineScript(t *testing.T) {
	server, capture := fakeJenkinsConfig(t)
	defer server.Close()
	service := connectM2(t, server, false)

	_, rpcErr := service.Handle(
		context.Background(),
		"jenkins/updateJob",
		json.RawMessage(`{"connectionId":"conn-1","job":"pipe","fields":{"script":"echo new"}}`),
	)
	if rpcErr != nil {
		t.Fatalf("updateJob error = %+v", rpcErr)
	}
	body := capture.body("POST /job/pipe/config.xml")
	if !strings.Contains(body, "<script>echo new</script>") {
		t.Errorf("posted body = %q", body)
	}
}

func TestRPCUpdateJobRename(t *testing.T) {
	server, capture := fakeJenkinsConfig(t)
	defer server.Close()
	service := connectM2(t, server, false)

	result, rpcErr := service.Handle(
		context.Background(),
		"jenkins/updateJob",
		json.RawMessage(`{"connectionId":"conn-1","job":"demo","fields":{"title":"demo-v2"}}`),
	)
	if rpcErr != nil {
		t.Fatalf("updateJob error = %+v", rpcErr)
	}
	payload := decodeResult[struct {
		FullName string `json:"fullName"`
	}](t, result)
	if payload.FullName != "demo-v2" {
		t.Errorf("fullName = %q, want demo-v2", payload.FullName)
	}
	if !capture.contains("doRename") {
		t.Errorf("doRename not called: %+v", capture.requests)
	}
}

func TestRPCCreateJob(t *testing.T) {
	server, capture := fakeJenkinsConfig(t)
	defer server.Close()
	service := connectM2(t, server, false)

	result, rpcErr := service.Handle(
		context.Background(),
		"jenkins/createJob",
		json.RawMessage(`{"connectionId":"conn-1","name":"new-job","kind":"pipeline","description":"nightly","script":"echo hi"}`),
	)
	if rpcErr != nil {
		t.Fatalf("createJob error = %+v", rpcErr)
	}
	payload := decodeResult[struct {
		Success  bool   `json:"success"`
		FullName string `json:"fullName"`
	}](t, result)
	if !payload.Success || payload.FullName != "new-job" {
		t.Errorf("result = %+v", payload)
	}
	if !capture.contains("createItem?name=new-job") {
		t.Errorf("createItem not called: %+v", capture.requests)
	}
	body := capture.body("POST /createItem")
	if !strings.Contains(body, "<flow-definition>") {
		t.Errorf("body = %q", body)
	}
}

func TestRPCDeleteJobAndView(t *testing.T) {
	server, capture := fakeJenkinsConfig(t)
	defer server.Close()
	service := connectM2(t, server, false)
	ctx := context.Background()

	if _, rpcErr := service.Handle(
		ctx,
		"jenkins/deleteJob",
		json.RawMessage(`{"connectionId":"conn-1","job":"demo"}`),
	); rpcErr != nil {
		t.Fatalf("deleteJob error = %+v", rpcErr)
	}
	if _, rpcErr := service.Handle(
		ctx,
		"jenkins/deleteView",
		json.RawMessage(`{"connectionId":"conn-1","view":"Builds"}`),
	); rpcErr != nil {
		t.Fatalf("deleteView error = %+v", rpcErr)
	}
	if !capture.contains("/job/demo/doDelete") {
		t.Errorf("job doDelete missing: %+v", capture.requests)
	}
	if !capture.contains("/view/Builds/doDelete") {
		t.Errorf("view doDelete missing: %+v", capture.requests)
	}
}

func TestRPCViewConfigAndUpdate(t *testing.T) {
	server, capture := fakeJenkinsConfig(t)
	defer server.Close()
	service := connectM2(t, server, false)
	ctx := context.Background()

	result, rpcErr := service.Handle(
		ctx,
		"jenkins/viewConfig",
		json.RawMessage(`{"connectionId":"conn-1","view":"Builds"}`),
	)
	if rpcErr != nil {
		t.Fatalf("viewConfig error = %+v", rpcErr)
	}
	payload := decodeResult[struct {
		XML    string `json:"xml"`
		Fields struct {
			Name     string   `json:"name"`
			Jobs     []string `json:"jobNames"`
			Recurse  bool     `json:"recurse"`
		} `json:"fields"`
	}](t, result)
	if !strings.Contains(payload.XML, "<name>Builds</name>") {
		t.Errorf("xml = %q", payload.XML)
	}
	if payload.Fields.Name != "Builds" || len(payload.Fields.Jobs) != 1 {
		t.Errorf("fields = %+v", payload.Fields)
	}

	// 前端按小驼峰取字段，这里检查序列化后的 JSON key，避免大小写不敏感解码掩盖问题。
	envelope, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T, want map[string]any", result)
	}
	encoded, err := json.Marshal(envelope["fields"])
	if err != nil {
		t.Fatalf("marshal fields: %v", err)
	}
	var fieldsMap map[string]any
	if err := json.Unmarshal(encoded, &fieldsMap); err != nil {
		t.Fatalf("unmarshal fields: %v", err)
	}
	for _, key := range []string{"name", "description", "jobNames", "includeRegex", "recurse"} {
		if _, exists := fieldsMap[key]; !exists {
			t.Errorf("fields missing lower camel key %q: %s", key, encoded)
		}
	}

	_, rpcErr = service.Handle(
		ctx,
		"jenkins/updateView",
		json.RawMessage(`{"connectionId":"conn-1","view":"Builds","fields":{"description":"new","jobNames":["api","web"]}}`),
	)
	if rpcErr != nil {
		t.Fatalf("updateView error = %+v", rpcErr)
	}
	body := capture.body("POST /view/Builds/config.xml")
	if !strings.Contains(body, "<description>new</description>") {
		t.Errorf("body = %q", body)
	}
	if !strings.Contains(body, "<string>api</string>") ||
		!strings.Contains(body, "<string>web</string>") {
		t.Errorf("jobs not replaced: %q", body)
	}
}

func TestRPCCreateView(t *testing.T) {
	server, capture := fakeJenkinsConfig(t)
	defer server.Close()
	service := connectM2(t, server, false)

	_, rpcErr := service.Handle(
		context.Background(),
		"jenkins/createView",
		json.RawMessage(`{"connectionId":"conn-1","name":"CRUD","description":"d","jobs":["demo"],"includeRegex":"^rel","recurse":true}`),
	)
	if rpcErr != nil {
		t.Fatalf("createView error = %+v", rpcErr)
	}
	if !capture.contains("/createView") {
		t.Errorf("createView not called: %+v", capture.requests)
	}
	body := capture.body("POST /createView")
	if !strings.Contains(body, "name=CRUD") {
		t.Errorf("body = %q", body)
	}
	if !strings.Contains(body, "json=") {
		t.Errorf("json missing: %q", body)
	}
}

func TestRPCUpdateViewRename(t *testing.T) {
	server, capture := fakeJenkinsConfig(t)
	defer server.Close()
	service := connectM2(t, server, false)

	result, rpcErr := service.Handle(
		context.Background(),
		"jenkins/updateView",
		json.RawMessage(`{"connectionId":"conn-1","view":"Builds","fields":{"name":"Builds2"}}`),
	)
	if rpcErr != nil {
		t.Fatalf("updateView error = %+v", rpcErr)
	}
	payload := decodeResult[struct {
		Success  bool   `json:"success"`
		FullName string `json:"fullName"`
	}](t, result)
	if !payload.Success || payload.FullName != "Builds2" {
		t.Errorf("result = %+v", payload)
	}
	if !capture.contains("/view/Builds/configSubmit") {
		t.Errorf("configSubmit not called: %+v", capture.requests)
	}
	if !capture.contains("POST /view/Builds2/config.xml") {
		t.Errorf("config not restored on the new view: %+v", capture.requests)
	}
}

func TestRPCUpdateViewRejectsDuplicateName(t *testing.T) {
	server, capture := fakeJenkinsConfig(t)
	defer server.Close()
	service := connectM2(t, server, false)

	_, rpcErr := service.Handle(
		context.Background(),
		"jenkins/updateView",
		json.RawMessage(`{"connectionId":"conn-1","view":"Builds","fields":{"name":"Builds"}}`),
	)
	if rpcErr != nil {
		t.Fatalf("same-name update should be a no-op, got %+v", rpcErr)
	}
	if capture.contains("/configSubmit") {
		t.Errorf("configSubmit should not be called for a same-name update: %+v", capture.requests)
	}
}

func TestRPCSaveJobConfig(t *testing.T) {
	server, capture := fakeJenkinsConfig(t)
	defer server.Close()
	service := connectM2(t, server, false)

	_, rpcErr := service.Handle(
		context.Background(),
		"jenkins/saveJob",
		json.RawMessage(`{"connectionId":"conn-1","job":"demo","xml":"<project><description>raw</description></project>"}`),
	)
	if rpcErr != nil {
		t.Fatalf("saveJob error = %+v", rpcErr)
	}
	body := capture.body("POST /job/demo/config.xml")
	if body != "<project><description>raw</description></project>" {
		t.Errorf("body = %q", body)
	}
}

func TestRPCConfigWritesRejectReadOnly(t *testing.T) {
	server, _ := fakeJenkinsConfig(t)
	defer server.Close()
	service := connectM2(t, server, true)
	ctx := context.Background()

	requests := []struct {
		method  string
		payload string
	}{
		{method: "jenkins/saveJob", payload: `{"connectionId":"conn-1","job":"demo","xml":"<project/>"}`},
		{method: "jenkins/createJob", payload: `{"connectionId":"conn-1","name":"x","kind":"freestyle"}`},
		{method: "jenkins/updateJob", payload: `{"connectionId":"conn-1","job":"demo","fields":{"description":"x"}}`},
		{method: "jenkins/renameJob", payload: `{"connectionId":"conn-1","job":"demo","newName":"x"}`},
		{method: "jenkins/deleteJob", payload: `{"connectionId":"conn-1","job":"demo"}`},
		{method: "jenkins/saveView", payload: `{"connectionId":"conn-1","view":"Builds","xml":"<hudson.model.ListView/>"}`},
		{method: "jenkins/createView", payload: `{"connectionId":"conn-1","name":"x"}`},
		{method: "jenkins/updateView", payload: `{"connectionId":"conn-1","view":"Builds","fields":{"description":"x"}}`},
		{method: "jenkins/deleteView", payload: `{"connectionId":"conn-1","view":"Builds"}`},
	}

	for _, request := range requests {
		t.Run(request.method, func(t *testing.T) {
			_, rpcErr := service.Handle(ctx, request.method, json.RawMessage(request.payload))
			if rpcErr == nil || rpcErr.Kind != "read_only" {
				t.Fatalf("error = %+v, want read_only", rpcErr)
			}
		})
	}
}
