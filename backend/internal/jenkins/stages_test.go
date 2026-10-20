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

func TestGetBuildStages(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		fmt.Fprint(w, `{
			"id":"35","name":"#35","status":"SUCCESS","durationMillis":60000,
			"stages":[
				{"id":"6","name":"获取构建人信息","status":"SUCCESS","startTimeMillis":1792398000000,"durationMillis":101,"pauseDurationMillis":0},
				{"id":"8","name":"打包构建","status":"FAILED","startTimeMillis":1792398000101,"durationMillis":3500,"pauseDurationMillis":0}
			]
		}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	stages, err := client.GetBuildStages(context.Background(), "team/demo", 35)
	if err != nil {
		t.Fatalf("GetBuildStages() error = %v", err)
	}
	if path != "/job/team/job/demo/35/wfapi/describe" {
		t.Errorf("path = %q, want wfapi/describe", path)
	}
	if len(stages) != 2 {
		t.Fatalf("stages = %d, want 2", len(stages))
	}
	if stages[0].Name != "获取构建人信息" || stages[0].Status != "success" || stages[0].DurationMillis != 101 {
		t.Errorf("stage 0 = %+v, want success 101ms", stages[0])
	}
	if stages[1].Status != "failed" {
		t.Errorf("stage 1 status = %q, want failed", stages[1].Status)
	}
}

func TestGetBuildStagesUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	_, err := client.GetBuildStages(context.Background(), "demo", 1)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrStagesUnavailable {
		t.Fatalf("error = %v, want stages_unavailable", err)
	}
}

func TestGetStageMatrix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/job/demo/api/json":
			if !strings.Contains(r.URL.Query().Get("tree"), "changeSet[items[commitId]]") {
				t.Errorf("tree missing changeSet: %s", r.URL.Query().Get("tree"))
			}
			fmt.Fprint(w, `{"builds":[
				{"number":35,"result":"SUCCESS","building":false,"timestamp":1792398000000,"duration":60000,"changeSet":{"items":[{"commitId":"a"}]}},
				{"number":34,"result":"FAILURE","building":false,"timestamp":1792397000000,"duration":50000,"changeSet":{"items":[{"commitId":"b"},{"commitId":"c"}]}},
				{"number":33,"result":"SUCCESS","building":false,"timestamp":1792396000000,"duration":40000,"changeSet":{"items":[]}}
			]}`)
		case r.URL.Path == "/job/demo/35/wfapi/describe":
			fmt.Fprint(w, `{"stages":[
				{"name":"A","status":"SUCCESS","durationMillis":100},
				{"name":"B","status":"SUCCESS","durationMillis":200}
			]}`)
		case r.URL.Path == "/job/demo/34/wfapi/describe":
			fmt.Fprint(w, `{"stages":[
				{"name":"A","status":"SUCCESS","durationMillis":80},
				{"name":"C","status":"SUCCESS","durationMillis":300}
			]}`)
		case r.URL.Path == "/job/demo/33/wfapi/describe":
			fmt.Fprint(w, `{"stages":[
				{"name":"A","status":"SUCCESS","durationMillis":60},
				{"name":"B","status":"SUCCESS","durationMillis":120},
				{"name":"C","status":"SUCCESS","durationMillis":0}
			]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	matrix, err := client.GetStageMatrix(context.Background(), "demo", 10)
	if err != nil {
		t.Fatalf("GetStageMatrix() error = %v", err)
	}
	if strings.Join(matrix.StageNames, ",") != "A,B,C" {
		t.Errorf("stage names = %v, want A,B,C", matrix.StageNames)
	}
	if len(matrix.Builds) != 3 {
		t.Fatalf("builds = %d, want 3", len(matrix.Builds))
	}
	if matrix.Builds[0].CommitCount != 1 || matrix.Builds[1].CommitCount != 2 || matrix.Builds[2].CommitCount != 0 {
		t.Errorf("commit counts = %d,%d,%d", matrix.Builds[0].CommitCount, matrix.Builds[1].CommitCount, matrix.Builds[2].CommitCount)
	}
	if stageByName(matrix.Builds[1], "B") != nil {
		t.Errorf("build 34 should not have stage B")
	}
	if got := averageByName(matrix, "A"); got != 80 {
		t.Errorf("average A = %d, want 80", got)
	}
	if got := averageByName(matrix, "B"); got != 160 {
		t.Errorf("average B = %d, want 160", got)
	}
	if got := averageByName(matrix, "C"); got != 300 {
		t.Errorf("average C = %d, want 300", got)
	}
	if matrix.AverageRun != 540 {
		t.Errorf("AverageRun = %d, want 540", matrix.AverageRun)
	}
}

func TestGetStageMatrixSkipsUnavailableRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/job/demo/api/json" {
			fmt.Fprint(w, `{"builds":[
				{"number":2,"result":"SUCCESS","building":false,"changeSet":{"items":[]}},
				{"number":1,"result":"SUCCESS","building":false,"changeSet":{"items":[]}}
			]}`)
			return
		}
		if r.URL.Path == "/job/demo/2/wfapi/describe" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"stages":[{"name":"Build","status":"SUCCESS","durationMillis":100}]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	matrix, err := client.GetStageMatrix(context.Background(), "demo", 10)
	if err != nil {
		t.Fatalf("GetStageMatrix() error = %v", err)
	}
	if len(matrix.Builds) != 2 || len(matrix.Builds[0].Stages) != 0 || len(matrix.Builds[1].Stages) != 1 {
		t.Fatalf("matrix = %+v, want one empty row and one Build row", matrix.Builds)
	}
}

func TestGetStageMatrixAllUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/job/demo/api/json" {
			fmt.Fprint(w, `{"builds":[{"number":1,"result":"SUCCESS","building":false}]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	_, err := client.GetStageMatrix(context.Background(), "demo", 10)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrStagesUnavailable {
		t.Fatalf("error = %v, want stages_unavailable", err)
	}
}

func stageByName(build StageBuild, name string) *Stage {
	for index := range build.Stages {
		if build.Stages[index].Name == name {
			return &build.Stages[index]
		}
	}
	return nil
}

func averageByName(matrix *StageMatrix, name string) int64 {
	for _, average := range matrix.Averages {
		if average.Name == name {
			return average.Duration
		}
	}
	return -1
}
