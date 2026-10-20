package jenkins

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetJobsRoot(t *testing.T) {
	var tree string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/json" {
			http.NotFound(w, r)
			return
		}
		tree = r.URL.Query().Get("tree")
		fmt.Fprint(w, `{"jobs":[
			{"_class":"hudson.model.FreeStyleProject","name":"build","fullName":"build","color":"red_anime","url":"http://ci/job/build/","lastBuild":{"number":7,"result":"FAILURE","timestamp":1700000000000,"duration":1234,"building":true}},
			{"_class":"com.cloudbees.hudson.plugins.folder.Folder","name":"team","fullName":"team","color":"blue","url":"http://ci/job/team/","lastBuild":null}
		]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	jobs, err := client.GetJobs(context.Background(), "")
	if err != nil {
		t.Fatalf("GetJobs() error = %v", err)
	}
	if !strings.Contains(tree, "lastBuild[number,result") {
		t.Errorf("tree = %q, want lastBuild fields", tree)
	}
	if len(jobs) != 2 {
		t.Fatalf("jobs = %d, want 2", len(jobs))
	}
	if jobs[0].Name != "team" || !jobs[0].IsFolder {
		t.Errorf("first job = %+v, want folder team first", jobs[0])
	}
	if jobs[1].Name != "build" || jobs[1].Status != StatusBuilding || !jobs[1].Building {
		t.Errorf("second job = %+v, want building build", jobs[1])
	}
}

func TestGetJobsFolder(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		fmt.Fprint(w, `{"jobs":[
			{"_class":"org.jenkinsci.plugins.workflow.job.WorkflowJob","name":"demo","color":"blue","url":"http://ci/job/team/job/demo/","lastBuild":{"number":1,"result":"SUCCESS","building":false}}
		]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	jobs, err := client.GetJobs(context.Background(), "team")
	if err != nil {
		t.Fatalf("GetJobs() error = %v", err)
	}
	if path != "/job/team/api/json" {
		t.Errorf("path = %q, want /job/team/api/json", path)
	}
	if len(jobs) != 1 || jobs[0].FullName != "team/demo" {
		t.Errorf("jobs = %+v, want fullName team/demo", jobs)
	}
	if jobs[0].Status != StatusSuccess {
		t.Errorf("status = %q, want success", jobs[0].Status)
	}
}

func TestGetJobParameters(t *testing.T) {
	var path, tree string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		tree = r.URL.Query().Get("tree")
		fmt.Fprint(w, `{
			"name":"demo","fullName":"team/demo","description":"Build demo","url":"http://ci/job/team/job/demo/",
			"color":"blue","buildable":true,"inQueue":false,
			"lastBuild":{"number":12,"result":"SUCCESS","timestamp":1700000000000,"duration":1000,"building":false},
			"lastSuccessfulBuild":{"number":12},"lastFailedBuild":{"number":9},
			"property":[{"parameterDefinitions":[
				{"name":"BRANCH","type":"StringParameterDefinition","description":"branch","defaultParameterValue":{"value":"main"}},
				{"name":"ENV","type":"ChoiceParameterDefinition","description":"env","choices":["dev","prod"]},
				{"name":"CLEAN","type":"BooleanParameterDefinition","defaultParameterValue":{"value":true}}
			]}]
		}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	job, err := client.GetJob(context.Background(), "team/demo")
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if path != "/job/team/job/demo/api/json" {
		t.Errorf("path = %q, want /job/team/job/demo/api/json", path)
	}
	if !strings.Contains(tree, "parameterDefinitions") {
		t.Errorf("tree = %q, want parameterDefinitions", tree)
	}
	if job.Status != StatusSuccess || job.LastBuild == nil || job.LastBuild.Number != 12 {
		t.Errorf("job = %+v, want success with build 12", job)
	}
	if len(job.Parameters) != 3 {
		t.Fatalf("parameters = %d, want 3", len(job.Parameters))
	}
	if job.Parameters[0].Name != "BRANCH" || job.Parameters[0].DefaultValue != "main" {
		t.Errorf("BRANCH = %+v, want default main", job.Parameters[0])
	}
	if len(job.Parameters[1].Choices) != 2 || job.Parameters[1].Choices[1] != "prod" {
		t.Errorf("ENV choices = %+v, want dev/prod", job.Parameters[1].Choices)
	}
	if job.Parameters[2].DefaultValue != "true" {
		t.Errorf("CLEAN default = %q, want true", job.Parameters[2].DefaultValue)
	}
}

func TestGetJobNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	_, err := client.GetJob(context.Background(), "missing")
	if err == nil {
		t.Fatal("GetJob() error = nil, want not found")
	}
}
