package jenkins

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetBuilds(t *testing.T) {
	var path, tree string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		tree = r.URL.Query().Get("tree")
		fmt.Fprint(w, `{"builds":[
			{"number":12,"result":"SUCCESS","building":false,"timestamp":1700000000000,"duration":1000,"displayName":"#12","actions":[{"causes":[{"shortDescription":"Started by user Alice","userId":"alice","userName":"Alice"}]}]},
			{"number":11,"result":null,"building":true,"timestamp":1699999999000,"duration":500,"displayName":"#11","actions":[]}
		]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	builds, err := client.GetBuilds(context.Background(), "team/demo", 5)
	if err != nil {
		t.Fatalf("GetBuilds() error = %v", err)
	}
	if path != "/job/team/job/demo/api/json" {
		t.Errorf("path = %q, want /job/team/job/demo/api/json", path)
	}
	if !strings.Contains(tree, "{0,5}") {
		t.Errorf("tree = %q, want range {0,5}", tree)
	}
	if len(builds) != 2 {
		t.Fatalf("builds = %d, want 2", len(builds))
	}
	if builds[0].Status != StatusSuccess || len(builds[0].Causes) != 1 {
		t.Errorf("build 12 = %+v, want success with one cause", builds[0])
	}
	if builds[0].Causes[0].ShortDescription != "Started by user Alice" {
		t.Errorf("cause = %+v, want user Alice", builds[0].Causes[0])
	}
	if builds[1].Status != StatusBuilding {
		t.Errorf("build 11 status = %q, want building", builds[1].Status)
	}
}

func TestGetBuildsKeepsChangesInternal(t *testing.T) {
	var tree string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tree = r.URL.Query().Get("tree")
		fmt.Fprint(w, `{"builds":[{"number":7,"result":"SUCCESS","building":false,"changeSet":{"items":[
			{"commitId":"a1","msg":"feat: 第一处","author":{"fullName":"Alice"}},
			{"commitId":"b2","msg":"fix: 第二处","author":{"fullName":"Bob"}}
		]}}]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	builds, err := client.GetBuilds(context.Background(), "demo", 5)
	if err != nil {
		t.Fatalf("GetBuilds() error = %v", err)
	}
	if !strings.Contains(tree, "changeSets[items[commitId,msg,author[fullName]]]") {
		t.Fatalf("tree missing changeSets details: %s", tree)
	}
	if len(builds) != 1 || builds[0].CommitCount != 2 {
		t.Fatalf("builds = %+v, want commitCount 2", builds)
	}
	if len(builds[0].Changes) != 2 || builds[0].Changes[1].Message != "fix: 第二处" {
		t.Errorf("changes = %+v, want two parsed entries", builds[0].Changes)
	}

	raw, err := json.Marshal(builds[0])
	if err != nil {
		t.Fatalf("marshal build: %v", err)
	}
	if strings.Contains(string(raw), `"changes"`) {
		t.Errorf("json = %s, want changes kept internal for jenkins/builds", raw)
	}
}

func TestGetBuildDetailReadsChangeSets(t *testing.T) {
	var tree string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tree = r.URL.Query().Get("tree")
		// Pipeline 构建详情：changelog 在 changeSets（复数）。
		fmt.Fprint(w, `{
			"number":2,"result":"SUCCESS","building":false,
			"changeSets":[{"items":[{"commitId":"945f1594aa","msg":"feat(export): 联调","author":{"fullName":"源哥"}}]}]
		}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	build, err := client.GetBuild(context.Background(), "demo", 2)
	if err != nil {
		t.Fatalf("GetBuild() error = %v", err)
	}
	if !strings.Contains(tree, "changeSets[items[commitId,msg,author[fullName]]]") {
		t.Fatalf("tree missing changeSets details: %s", tree)
	}
	if len(build.ChangeSet) != 1 {
		t.Fatalf("changeSet = %d, want 1", len(build.ChangeSet))
	}
	if build.ChangeSet[0].CommitID != "945f1594aa" || build.ChangeSet[0].Author != "源哥" {
		t.Errorf("change = %+v", build.ChangeSet[0])
	}
}

func TestGetBuildDetail(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		fmt.Fprint(w, `{
			"number":12,"result":"FAILURE","building":false,"timestamp":1700000000000,
			"duration":1000,"estimatedDuration":2000,"displayName":"#12",
			"fullDisplayName":"team » demo #12","url":"http://ci/job/team/job/demo/12/",
			"actions":[{"causes":[{"shortDescription":"Started by timer"}]}],
			"changeSet":{"items":[{"commitId":"abc123","msg":"fix bug","author":{"fullName":"Bob"}}]},
			"artifacts":[{"fileName":"app.jar","relativePath":"target/app.jar"}]
		}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	build, err := client.GetBuild(context.Background(), "team/demo", 12)
	if err != nil {
		t.Fatalf("GetBuild() error = %v", err)
	}
	if path != "/job/team/job/demo/12/api/json" {
		t.Errorf("path = %q, want /job/team/job/demo/12/api/json", path)
	}
	if build.Status != StatusFailure || build.EstimatedDuration != 2000 {
		t.Errorf("build = %+v, want failure with estimate 2000", build)
	}
	if len(build.Causes) != 1 || build.Causes[0].ShortDescription != "Started by timer" {
		t.Errorf("causes = %+v, want timer", build.Causes)
	}
	if len(build.ChangeSet) != 1 || build.ChangeSet[0].CommitID != "abc123" {
		t.Errorf("changeSet = %+v, want abc123", build.ChangeSet)
	}
	if len(build.Artifacts) != 1 || build.Artifacts[0].FileName != "app.jar" {
		t.Errorf("artifacts = %+v, want app.jar", build.Artifacts)
	}
}

func TestGetBuildsDefaultLimit(t *testing.T) {
	var tree string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tree = r.URL.Query().Get("tree")
		fmt.Fprint(w, `{"builds":[]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	if _, err := client.GetBuilds(context.Background(), "demo", 0); err != nil {
		t.Fatalf("GetBuilds() error = %v", err)
	}
	if !strings.Contains(tree, "{0,20}") {
		t.Errorf("tree = %q, want default range {0,20}", tree)
	}
}
