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

func fakeJenkinsM2(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/whoAmI/api/json", func(w http.ResponseWriter, r *http.Request) {
		if user, pass, ok := r.BasicAuth(); !ok || user != "alice" || pass != "token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("X-Jenkins", "2.440.3")
		fmt.Fprint(w, `{"name":"alice","authenticated":true}`)
	})
	mux.HandleFunc("/crumbIssuer/api/json", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	mux.HandleFunc("/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jobs":[
			{"_class":"com.cloudbees.hudson.plugins.folder.Folder","name":"team","fullName":"team","color":"blue","url":"http://ci/job/team/"},
			{"_class":"hudson.model.FreeStyleProject","name":"demo","fullName":"demo","color":"red","url":"http://ci/job/demo/","lastBuild":{"number":3,"result":"FAILURE","building":false}}
		]}`)
	})
	mux.HandleFunc("/job/team/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"jobs":[{"_class":"org.jenkinsci.plugins.workflow.job.WorkflowJob","name":"demo","fullName":"team/demo","color":"blue","url":"http://ci/job/team/job/demo/","lastBuild":{"number":12,"result":"SUCCESS","building":false}}]}`)
	})
	mux.HandleFunc("/job/team/job/demo/api/json", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Query().Get("tree"), "builds[") {
			fmt.Fprint(w, `{"builds":[
				{"number":12,"result":"SUCCESS","building":false,"timestamp":1700000000000,"duration":1000,"displayName":"#12","actions":[{"causes":[{"shortDescription":"Started by user Alice"}]}]},
				{"number":11,"result":"FAILURE","building":false,"timestamp":1699999999000,"duration":800,"displayName":"#11","actions":[]}
			]}`)
			return
		}
		fmt.Fprint(w, `{
			"name":"demo","fullName":"team/demo","description":"Build demo","color":"blue","buildable":true,
			"lastBuild":{"number":12,"result":"SUCCESS","building":false},
			"property":[{"parameterDefinitions":[
				{"name":"BRANCH","type":"StringParameterDefinition","defaultParameterValue":{"value":"main"}},
				{"name":"ENV","type":"ChoiceParameterDefinition","choices":["dev","prod"]}
			]}]
		}`)
	})
	mux.HandleFunc("/job/team/job/demo/12/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"number":12,"result":"SUCCESS","building":false,"timestamp":1700000000000,"duration":1000,
			"actions":[{"causes":[{"shortDescription":"Started by user Alice"}]}],
			"changeSet":{"items":[{"commitId":"abc","msg":"fix bug","author":{"fullName":"Bob"}}]},
			"artifacts":[{"fileName":"app.jar","relativePath":"target/app.jar"}]}`)
	})
	mux.HandleFunc("/job/team/job/demo/12/logText/progressiveText", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("start") {
		case "0":
			w.Header().Set("X-Text-Size", "6")
			w.Header().Set("X-More-Data", "true")
			fmt.Fprint(w, "hello\n")
		default:
			w.Header().Set("X-Text-Size", "12")
			w.Header().Set("X-More-Data", "false")
			fmt.Fprint(w, "world\n")
		}
	})
	triggerHandler := func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		if r.Form.Get("BRANCH") != "main" {
			t.Errorf("BRANCH = %q, want main", r.Form.Get("BRANCH"))
		}
		w.Header().Set("Location", "/queue/item/42/")
		w.WriteHeader(http.StatusCreated)
	}
	mux.HandleFunc("/job/team/job/demo/build", triggerHandler)
	mux.HandleFunc("/job/team/job/demo/buildWithParameters", triggerHandler)
	mux.HandleFunc("/job/team/job/demo/12/stop", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return httptest.NewServer(mux)
}

func connectM2(t *testing.T, server *httptest.Server, readOnly bool) *Service {
	t.Helper()
	service := NewService()
	payload := lifecyclePayload(t, server.URL, "token", "api_token", "", readOnly)
	if _, rpcErr := service.Handle(context.Background(), "connection/connect", payload); rpcErr != nil {
		t.Fatalf("connect error = %+v", rpcErr)
	}
	return service
}

func TestRPCJobsAndJob(t *testing.T) {
	server := fakeJenkinsM2(t)
	defer server.Close()
	service := connectM2(t, server, true)
	ctx := context.Background()

	result, rpcErr := service.Handle(ctx, "jenkins/jobs", json.RawMessage(`{"connectionId":"conn-1"}`))
	if rpcErr != nil {
		t.Fatalf("jobs error = %+v", rpcErr)
	}
	decoded := decodeResult[struct {
		Jobs []struct {
			Name     string `json:"name"`
			IsFolder bool   `json:"isFolder"`
			Status   string `json:"status"`
		} `json:"jobs"`
	}](t, result)
	if len(decoded.Jobs) != 2 || decoded.Jobs[0].Name != "team" || !decoded.Jobs[0].IsFolder {
		t.Errorf("jobs = %+v, want folder team first", decoded.Jobs)
	}
	if decoded.Jobs[1].Status != "failure" {
		t.Errorf("demo status = %q, want failure", decoded.Jobs[1].Status)
	}

	result, rpcErr = service.Handle(ctx, "jenkins/jobs", json.RawMessage(`{"connectionId":"conn-1","folder":"team"}`))
	if rpcErr != nil {
		t.Fatalf("folder jobs error = %+v", rpcErr)
	}
	decodedFolder := decodeResult[struct {
		Jobs []struct {
			FullName string `json:"fullName"`
		} `json:"jobs"`
	}](t, result)
	if len(decodedFolder.Jobs) != 1 || decodedFolder.Jobs[0].FullName != "team/demo" {
		t.Errorf("folder jobs = %+v, want team/demo", decodedFolder.Jobs)
	}

	result, rpcErr = service.Handle(ctx, "jenkins/job", json.RawMessage(`{"connectionId":"conn-1","job":"team/demo"}`))
	if rpcErr != nil {
		t.Fatalf("job error = %+v", rpcErr)
	}
	job := decodeResult[struct {
		FullName   string `json:"fullName"`
		Parameters []struct {
			Name         string `json:"name"`
			DefaultValue string `json:"defaultValue"`
		} `json:"parameters"`
	}](t, result)
	if job.FullName != "team/demo" || len(job.Parameters) != 2 || job.Parameters[0].DefaultValue != "main" {
		t.Errorf("job = %+v, want team/demo with BRANCH=main", job)
	}
}

func TestRPCBuildsAndBuild(t *testing.T) {
	server := fakeJenkinsM2(t)
	defer server.Close()
	service := connectM2(t, server, true)
	ctx := context.Background()

	result, rpcErr := service.Handle(ctx, "jenkins/builds", json.RawMessage(`{"connectionId":"conn-1","job":"team/demo","limit":5}`))
	if rpcErr != nil {
		t.Fatalf("builds error = %+v", rpcErr)
	}
	builds := decodeResult[struct {
		Builds []struct {
			Number int    `json:"number"`
			Status string `json:"status"`
			Causes []struct {
				ShortDescription string `json:"shortDescription"`
			} `json:"causes"`
		} `json:"builds"`
	}](t, result)
	if len(builds.Builds) != 2 || builds.Builds[0].Status != "success" {
		t.Errorf("builds = %+v, want 2 with success first", builds.Builds)
	}
	if builds.Builds[0].Causes[0].ShortDescription != "Started by user Alice" {
		t.Errorf("causes = %+v, want Alice", builds.Builds[0].Causes)
	}

	result, rpcErr = service.Handle(ctx, "jenkins/build", json.RawMessage(`{"connectionId":"conn-1","job":"team/demo","number":12}`))
	if rpcErr != nil {
		t.Fatalf("build error = %+v", rpcErr)
	}
	build := decodeResult[struct {
		Status    string `json:"status"`
		ChangeSet []struct {
			CommitID string `json:"commitId"`
		} `json:"changeSet"`
		Artifacts []struct {
			FileName string `json:"fileName"`
		} `json:"artifacts"`
	}](t, result)
	if build.Status != "success" || len(build.ChangeSet) != 1 || build.ChangeSet[0].CommitID != "abc" {
		t.Errorf("build = %+v, want success with abc changeset", build)
	}
	if len(build.Artifacts) != 1 || build.Artifacts[0].FileName != "app.jar" {
		t.Errorf("artifacts = %+v, want app.jar", build.Artifacts)
	}
}

func TestRPCConsole(t *testing.T) {
	server := fakeJenkinsM2(t)
	defer server.Close()
	service := connectM2(t, server, true)
	ctx := context.Background()

	result, rpcErr := service.Handle(ctx, "jenkins/console", json.RawMessage(`{"connectionId":"conn-1","job":"team/demo","number":12,"start":0}`))
	if rpcErr != nil {
		t.Fatalf("console error = %+v", rpcErr)
	}
	chunk := decodeResult[struct {
		Text      string `json:"text"`
		NextStart int64  `json:"nextStart"`
		MoreData  bool   `json:"moreData"`
	}](t, result)
	if chunk.Text != "hello\n" || chunk.NextStart != 6 || !chunk.MoreData {
		t.Errorf("chunk = %+v, want hello/6/more", chunk)
	}

	result, rpcErr = service.Handle(ctx, "jenkins/console", json.RawMessage(`{"connectionId":"conn-1","job":"team/demo","number":12,"start":6}`))
	if rpcErr != nil {
		t.Fatalf("console 2 error = %+v", rpcErr)
	}
	chunk = decodeResult[struct {
		Text      string `json:"text"`
		NextStart int64  `json:"nextStart"`
		MoreData  bool   `json:"moreData"`
	}](t, result)
	if chunk.Text != "world\n" || chunk.NextStart != 12 || chunk.MoreData {
		t.Errorf("chunk2 = %+v, want world/12/done", chunk)
	}
}

func TestRPCTriggerAndStop(t *testing.T) {
	server := fakeJenkinsM2(t)
	defer server.Close()
	service := connectM2(t, server, false)
	ctx := context.Background()

	result, rpcErr := service.Handle(ctx, "jenkins/trigger", json.RawMessage(`{"connectionId":"conn-1","job":"team/demo","params":{"BRANCH":"main"}}`))
	if rpcErr != nil {
		t.Fatalf("trigger error = %+v", rpcErr)
	}
	trigger := decodeResult[struct {
		Success bool `json:"success"`
		QueueID int  `json:"queueId"`
	}](t, result)
	if !trigger.Success || trigger.QueueID != 42 {
		t.Errorf("trigger = %+v, want queue 42", trigger)
	}

	if _, rpcErr := service.Handle(ctx, "jenkins/stop", json.RawMessage(`{"connectionId":"conn-1","job":"team/demo","number":12}`)); rpcErr != nil {
		t.Fatalf("stop error = %+v", rpcErr)
	}
}

func TestRPCTriggerReadOnly(t *testing.T) {
	server := fakeJenkinsM2(t)
	defer server.Close()
	service := connectM2(t, server, true)

	_, rpcErr := service.Handle(context.Background(), "jenkins/trigger", json.RawMessage(`{"connectionId":"conn-1","job":"team/demo"}`))
	if rpcErr == nil || rpcErr.Kind != "read_only" {
		t.Fatalf("error = %+v, want read_only", rpcErr)
	}
}

func TestRPCJobsMissingParameter(t *testing.T) {
	server := fakeJenkinsM2(t)
	defer server.Close()
	service := connectM2(t, server, true)

	_, rpcErr := service.Handle(context.Background(), "jenkins/job", json.RawMessage(`{"connectionId":"conn-1"}`))
	if rpcErr == nil || rpcErr.Code != -32602 {
		t.Fatalf("error = %+v, want invalid params", rpcErr)
	}
}
