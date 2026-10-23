package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func fakeJenkinsStages(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/whoAmI/api/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Jenkins", "2.550")
		fmt.Fprint(w, `{"name":"alice","authenticated":true}`)
	})
	mux.HandleFunc("/crumbIssuer/api/json", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	mux.HandleFunc("/job/demo-pipeline/api/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"builds":[
			{"number":35,"result":"SUCCESS","building":false,"timestamp":1792398000000,"duration":60000,"changeSet":{"items":[{"commitId":"a"}]}},
			{"number":34,"result":"SUCCESS","building":false,"timestamp":1792397000000,"duration":50000,"changeSet":{"items":[{"commitId":"b"},{"commitId":"c"}]}}
		]}`)
	})
	mux.HandleFunc("/job/demo-pipeline/35/wfapi/describe", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"stages":[
			{"name":"获取构建人信息","status":"SUCCESS","durationMillis":85},
			{"name":"打包构建","status":"FAILED","durationMillis":120000}
		]}`)
	})
	mux.HandleFunc("/job/demo-pipeline/34/wfapi/describe", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"stages":[
			{"name":"获取构建人信息","status":"SUCCESS","durationMillis":75},
			{"name":"打包构建","status":"FAILED","durationMillis":60000}
		]}`)
	})
	return httptest.NewServer(mux)
}

func TestRPCStages(t *testing.T) {
	server := fakeJenkinsStages(t)
	defer server.Close()
	service := connectM2(t, server, true)

	result, rpcErr := service.Handle(
		context.Background(),
		"jenkins/stages",
		json.RawMessage(`{"connectionId":"conn-1","job":"demo-pipeline","limit":5}`),
	)
	if rpcErr != nil {
		t.Fatalf("stages error = %+v", rpcErr)
	}
	matrix := decodeResult[struct {
		Job        string   `json:"job"`
		StageNames []string `json:"stageNames"`
		Builds     []struct {
			Number      int `json:"number"`
			CommitCount int `json:"commitCount"`
			Stages      []struct {
				Name           string `json:"name"`
				Status         string `json:"status"`
				DurationMillis int64  `json:"durationMillis"`
			} `json:"stages"`
		} `json:"builds"`
		Averages []struct {
			Name     string `json:"name"`
			Duration int64  `json:"duration"`
		} `json:"averages"`
		AverageRun int64 `json:"averageRun"`
	}](t, result)

	if matrix.Job != "demo-pipeline" || len(matrix.StageNames) != 2 {
		t.Fatalf("matrix = %+v, want 2 stages", matrix)
	}
	if len(matrix.Builds) != 2 || matrix.Builds[0].CommitCount != 1 {
		t.Fatalf("builds = %+v, want 2 with commit count", matrix.Builds)
	}
	if matrix.Builds[1].Stages[1].Status != "failed" {
		t.Errorf("build 34 stage status = %q, want failed", matrix.Builds[1].Stages[1].Status)
	}
	if matrix.Averages[0].Duration != 80 {
		t.Errorf("average stage 0 = %d, want 80", matrix.Averages[0].Duration)
	}
	if matrix.AverageRun != 80 {
		t.Errorf("AverageRun = %d, want 80 (failed stage excluded)", matrix.AverageRun)
	}
}

func TestRPCStagesMissingJob(t *testing.T) {
	server := fakeJenkinsStages(t)
	defer server.Close()
	service := connectM2(t, server, true)

	_, rpcErr := service.Handle(context.Background(), "jenkins/stages", json.RawMessage(`{"connectionId":"conn-1"}`))
	if rpcErr == nil || rpcErr.Kind != "invalid_params" {
		t.Fatalf("error = %+v, want invalid_params", rpcErr)
	}
}

// 构建列表的进度条按阶段步进：只查正在跑的那一次构建，别拉整张矩阵。
func TestRPCBuildStages(t *testing.T) {
	server := fakeJenkinsStages(t)
	defer server.Close()
	service := connectM2(t, server, true)

	result, rpcErr := service.Handle(
		context.Background(),
		"jenkins/buildStages",
		json.RawMessage(`{"connectionId":"conn-1","job":"demo-pipeline","number":35}`),
	)
	if rpcErr != nil {
		t.Fatalf("buildStages error = %+v", rpcErr)
	}
	payload := decodeResult[struct {
		Stages []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"stages"`
	}](t, result)
	if len(payload.Stages) != 2 {
		t.Fatalf("stages = %+v, want 2", payload.Stages)
	}
	if payload.Stages[1].Name != "打包构建" || payload.Stages[1].Status != "failed" {
		t.Errorf("stage = %+v, want 打包构建/failed", payload.Stages[1])
	}
}

// 未注册的构建号 → wfapi/describe 404 → stages_unavailable，前端据此回退时长估算。
func TestRPCBuildStagesUnavailable(t *testing.T) {
	server := fakeJenkinsStages(t)
	defer server.Close()
	service := connectM2(t, server, true)

	_, rpcErr := service.Handle(
		context.Background(),
		"jenkins/buildStages",
		json.RawMessage(`{"connectionId":"conn-1","job":"demo-pipeline","number":999}`),
	)
	if rpcErr == nil || rpcErr.Kind != "stages_unavailable" {
		t.Fatalf("error = %+v, want stages_unavailable", rpcErr)
	}
}

func TestRPCBuildStagesBadNumber(t *testing.T) {
	server := fakeJenkinsStages(t)
	defer server.Close()
	service := connectM2(t, server, true)

	_, rpcErr := service.Handle(
		context.Background(),
		"jenkins/buildStages",
		json.RawMessage(`{"connectionId":"conn-1","job":"demo-pipeline","number":0}`),
	)
	if rpcErr == nil || rpcErr.Kind != "invalid_params" {
		t.Fatalf("error = %+v, want invalid_params", rpcErr)
	}
}
