package jenkins

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestParseJobFields(t *testing.T) {
	freeFields, err := ParseJobFields(jobConfigFixture, "team/demo")
	if err != nil {
		t.Fatalf("ParseJobFields(free) error = %v", err)
	}
	if freeFields.Kind != string(JobKindFreeStyle) {
		t.Errorf("kind = %q, want freestyle", freeFields.Kind)
	}
	if freeFields.Title != "demo" {
		t.Errorf("title = %q, want demo", freeFields.Title)
	}
	if freeFields.Description != "old description" {
		t.Errorf("description = %q", freeFields.Description)
	}
	if freeFields.Disabled {
		t.Error("disabled = true, want false")
	}
	if freeFields.Script != "" {
		t.Errorf("script = %q, want empty", freeFields.Script)
	}

	pipelineFields, err := ParseJobFields(pipelineConfigFixture, "pipe")
	if err != nil {
		t.Fatalf("ParseJobFields(pipeline) error = %v", err)
	}
	if pipelineFields.Kind != string(JobKindPipeline) {
		t.Errorf("kind = %q, want pipeline", pipelineFields.Kind)
	}
	if pipelineFields.Script != "echo old" {
		t.Errorf("script = %q, want echo old", pipelineFields.Script)
	}
}

func TestParseListViewFields(t *testing.T) {
	cfg, err := ParseListViewFields(listViewConfigFixture)
	if err != nil {
		t.Fatalf("ParseListViewFields() error = %v", err)
	}
	if cfg.Name != "Builds" {
		t.Errorf("name = %q", cfg.Name)
	}
	if cfg.Description != "old description" {
		t.Errorf("description = %q", cfg.Description)
	}
	if len(cfg.JobNames) != 1 || cfg.JobNames[0] != "demo" {
		t.Errorf("jobNames = %+v, want [demo]", cfg.JobNames)
	}
	if cfg.IncludeRegex != "" {
		t.Errorf("includeRegex = %q, want empty", cfg.IncludeRegex)
	}
	if cfg.Recurse {
		t.Error("recurse = true, want false")
	}
}

func TestUpdateJobDescriptionPreservesConfig(t *testing.T) {
	var (
		posted     string
		saveCalls  atomic.Int32
		getCalls   atomic.Int32
		renameHits atomic.Int32
	)
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.EscapedPath() == "/job/demo/config.xml" && r.Method == http.MethodGet:
			getCalls.Add(1)
			fmt.Fprint(w, jobConfigFixture)
		case r.URL.EscapedPath() == "/job/demo/config.xml" && r.Method == http.MethodPost:
			saveCalls.Add(1)
			body, _ := io.ReadAll(r.Body)
			posted = string(body)
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "doRename"):
			renameHits.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	fullName, err := client.UpdateJob(context.Background(), "demo", JobUpdate{
		Description:    "updated",
		SetDescription: true,
	})
	if err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}
	if fullName != "demo" {
		t.Errorf("fullName = %q, want demo", fullName)
	}
	if getCalls.Load() != 1 || saveCalls.Load() != 1 {
		t.Errorf("get/save calls = %d/%d, want 1/1", getCalls.Load(), saveCalls.Load())
	}
	if renameHits.Load() != 0 {
		t.Errorf("rename calls = %d, want 0", renameHits.Load())
	}
	if !strings.Contains(posted, "<description>updated</description>") {
		t.Errorf("description not updated: %s", posted)
	}
	if !strings.Contains(posted, "hudson.plugins.git.GitSCM") {
		t.Errorf("SCM dropped: %s", posted)
	}
}

func TestUpdateJobRenamesBeforeSaving(t *testing.T) {
	var (
		renamedTo string
		savedPath string
	)
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.EscapedPath() == "/job/demo/doRename":
			renamedTo = r.URL.Query().Get("newName")
			w.WriteHeader(http.StatusOK)
		case r.URL.EscapedPath() == "/job/demo-v2/config.xml" && r.Method == http.MethodGet:
			fmt.Fprint(w, jobConfigFixture)
		case r.URL.EscapedPath() == "/job/demo-v2/config.xml" && r.Method == http.MethodPost:
			savedPath = r.URL.EscapedPath()
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	fullName, err := client.UpdateJob(context.Background(), "demo", JobUpdate{
		Title:        "demo-v2",
		SetTitle:     true,
		Disabled:     true,
		SetDisabled:  true,
	})
	if err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}
	if fullName != "demo-v2" {
		t.Errorf("fullName = %q, want demo-v2", fullName)
	}
	if renamedTo != "demo-v2" {
		t.Errorf("renamedTo = %q", renamedTo)
	}
	if savedPath != "/job/demo-v2/config.xml" {
		t.Errorf("saved path = %q", savedPath)
	}
}

func TestUpdateJobPipelineScript(t *testing.T) {
	var posted string
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/job/pipe/config.xml" {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodGet {
			fmt.Fprint(w, pipelineConfigFixture)
			return
		}
		body, _ := io.ReadAll(r.Body)
		posted = string(body)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if _, err := client.UpdateJob(context.Background(), "pipe", JobUpdate{
		Script:    "echo updated",
		SetScript: true,
	}); err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}
	if !strings.Contains(posted, "<script>echo updated</script>") {
		t.Errorf("script not updated: %s", posted)
	}
	if !strings.Contains(posted, "<sandbox>true</sandbox>") {
		t.Errorf("sandbox dropped: %s", posted)
	}
}

func TestUpdateJobWithoutFieldsSkipsRequests(t *testing.T) {
	var calls atomic.Int32
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	fullName, err := client.UpdateJob(context.Background(), "demo", JobUpdate{})
	if err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}
	if fullName != "demo" {
		t.Errorf("fullName = %q", fullName)
	}
	if calls.Load() != 0 {
		t.Errorf("calls = %d, want 0", calls.Load())
	}
}

func TestUpdateJobRejectsReadOnly(t *testing.T) {
	var calls atomic.Int32
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	err := func() error {
		_, err := client.UpdateJob(context.Background(), "demo", JobUpdate{
			Description:    "x",
			SetDescription: true,
		})
		return err
	}()
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrReadOnly {
		t.Fatalf("error = %v, want read_only", err)
	}
	if calls.Load() != 0 {
		t.Errorf("calls = %d, want 0", calls.Load())
	}
}

func TestUpdateViewKeepsUnchangedFields(t *testing.T) {
	var posted string
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/view/Builds/config.xml" {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodGet {
			fmt.Fprint(w, listViewConfigFixture)
			return
		}
		body, _ := io.ReadAll(r.Body)
		posted = string(body)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if _, err := client.UpdateView(context.Background(), "Builds", ViewUpdate{
		Description:    "new description",
		SetDescription: true,
	}); err != nil {
		t.Fatalf("UpdateView() error = %v", err)
	}
	if !strings.Contains(posted, "<description>new description</description>") {
		t.Errorf("description not updated: %s", posted)
	}
	if !strings.Contains(posted, "<string>demo</string>") {
		t.Errorf("job names were dropped: %s", posted)
	}
	if !strings.Contains(posted, "<hudson.views.StatusColumn/>") {
		t.Errorf("columns dropped: %s", posted)
	}
}

func TestUpdateViewReplacesJobs(t *testing.T) {
	var posted string
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/view/Builds/config.xml" {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodGet {
			fmt.Fprint(w, listViewConfigFixture)
			return
		}
		body, _ := io.ReadAll(r.Body)
		posted = string(body)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if _, err := client.UpdateView(context.Background(), "Builds", ViewUpdate{
		JobNames:    []string{"api", "web"},
		SetJobNames: true,
	}); err != nil {
		t.Fatalf("UpdateView() error = %v", err)
	}
	if !strings.Contains(posted, "<string>api</string>") ||
		!strings.Contains(posted, "<string>web</string>") {
		t.Errorf("job names not replaced: %s", posted)
	}
	if strings.Contains(posted, "<string>demo</string>") {
		t.Errorf("old job names still present: %s", posted)
	}
	if !strings.Contains(posted, "<description>old description</description>") {
		t.Errorf("description was lost: %s", posted)
	}
}

func TestUpdateViewRejectsReadOnly(t *testing.T) {
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	_, err := client.UpdateView(context.Background(), "Builds", ViewUpdate{
		Description:    "x",
		SetDescription: true,
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrReadOnly {
		t.Fatalf("error = %v, want read_only", err)
	}
}

const allViewConfigFixture = `<?xml version='1.1' encoding='UTF-8'?>
<hudson.model.AllView>
  <name>all</name>
  <filterExecutors>false</filterExecutors>
  <filterQueue>false</filterQueue>
  <properties class="hudson.model.View$PropertyList"/>
</hudson.model.AllView>`

// TestRenameViewPostsFullForm 覆盖改名接口的真实形态：
// Jenkins 的 configSubmit 会全量重建视图，表单不完整就会丢任务与列（实测见
// docs/plan-view-rename.md 实验 A）。
func TestRenameViewPostsFullForm(t *testing.T) {
	var (
		submitted  url.Values
		restoredTo string
		restoreXML string
	)
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/json" && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"views":[{"name":"Builds"},{"name":"Ops"}]}`)
		case r.URL.EscapedPath() == "/view/Builds/config.xml" && r.Method == http.MethodGet:
			fmt.Fprint(w, listViewConfigFixture)
		case r.URL.EscapedPath() == "/view/Builds/configSubmit" && r.Method == http.MethodPost:
			if err := r.ParseForm(); err != nil {
				t.Errorf("ParseForm() error = %v", err)
			}
			submitted = r.PostForm
			w.WriteHeader(http.StatusFound)
		case r.URL.EscapedPath() == "/view/Builds2/config.xml" && r.Method == http.MethodPost:
			restoredTo = r.URL.EscapedPath()
			body, _ := io.ReadAll(r.Body)
			restoreXML = string(body)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if err := client.RenameView(context.Background(), "Builds", "Builds2"); err != nil {
		t.Fatalf("RenameView() error = %v", err)
	}

	if got := submitted.Get("name"); got != "Builds2" {
		t.Errorf("form name = %q, want Builds2", got)
	}
	if got := submitted.Get("item_demo"); got != "on" {
		t.Errorf("form item_demo = %q, want on", got)
	}
	payload := submitted.Get("json")
	for _, want := range []string{
		`"description":"old description"`,
		`"recurse":false`,
		`"filterQueue":false`,
		`"filterExecutors":false`,
		`"kind":"hudson.views.StatusColumn"`,
		`"kind":"hudson.views.JobColumn"`,
	} {
		if !strings.Contains(payload, want) {
			t.Errorf("json payload missing %s: %s", want, payload)
		}
	}

	// configSubmit 会清空 columns/properties，必须用原 XML 再回写一次。
	if restoredTo != "/view/Builds2/config.xml" {
		t.Errorf("restore path = %q, want /view/Builds2/config.xml", restoredTo)
	}
	if !strings.Contains(restoreXML, "<hudson.views.StatusColumn/>") ||
		!strings.Contains(restoreXML, "<string>demo</string>") {
		t.Errorf("restored XML lost view config:\n%s", restoreXML)
	}
}

func TestRenameViewIncludesRegexInForm(t *testing.T) {
	var submitted url.Values
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/json" && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"views":[{"name":"Builds"}]}`)
		case r.URL.EscapedPath() == "/view/Builds/config.xml" && r.Method == http.MethodGet:
			fmt.Fprint(w, strings.Replace(listViewConfigFixture,
				"<recurse>false</recurse>",
				"<includeRegex>release-.*</includeRegex>\n  <recurse>true</recurse>", 1))
		case r.URL.EscapedPath() == "/view/Builds/configSubmit" && r.Method == http.MethodPost:
			if err := r.ParseForm(); err != nil {
				t.Errorf("ParseForm() error = %v", err)
			}
			submitted = r.PostForm
		case r.URL.EscapedPath() == "/view/Builds2/config.xml" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if err := client.RenameView(context.Background(), "Builds", "Builds2"); err != nil {
		t.Fatalf("RenameView() error = %v", err)
	}
	if submitted.Get("useincluderegex") == "" {
		t.Error("useincluderegex missing: Jenkins would drop includeRegex")
	}
	if got := submitted.Get("includeRegex"); got != "release-.*" {
		t.Errorf("includeRegex = %q, want release-.*", got)
	}
	if !strings.Contains(submitted.Get("json"), `"recurse":true`) {
		t.Errorf("recurse not carried in json: %s", submitted.Get("json"))
	}
}

func TestRenameViewRejectsDuplicateName(t *testing.T) {
	var posts atomic.Int64
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/json" && r.Method == http.MethodGet {
			fmt.Fprint(w, `{"views":[{"name":"Builds"},{"name":"Ops"}]}`)
			return
		}
		if r.Method == http.MethodPost {
			posts.Add(1)
		}
		http.NotFound(w, r)
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	err := client.RenameView(context.Background(), "Builds", "Ops")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrInvalidConfig {
		t.Fatalf("error = %v, want invalid_config", err)
	}
	if !strings.Contains(apiErr.Message, "already exists") {
		t.Errorf("message = %q, want already exists", apiErr.Message)
	}
	if posts.Load() != 0 {
		t.Errorf("posts = %d, want 0", posts.Load())
	}
}

func TestRenameViewRejectsNonListView(t *testing.T) {
	var posts atomic.Int64
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/json" && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"views":[{"name":"all"}]}`)
		case r.URL.EscapedPath() == "/view/all/config.xml" && r.Method == http.MethodGet:
			fmt.Fprint(w, allViewConfigFixture)
		default:
			if r.Method == http.MethodPost {
				posts.Add(1)
			}
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	err := client.RenameView(context.Background(), "all", "all2")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrInvalidConfig {
		t.Fatalf("error = %v, want invalid_config", err)
	}
	if !strings.Contains(apiErr.Message, "not a ListView") {
		t.Errorf("message = %q, want not a ListView", apiErr.Message)
	}
	if posts.Load() != 0 {
		t.Errorf("posts = %d, want 0", posts.Load())
	}
}

func TestUpdateViewRenamesThenPatchesNewPath(t *testing.T) {
	var (
		submitted  url.Values
		finalPath  string
		finalBody  string
		newPathGet int
	)
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/json" && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"views":[{"name":"Builds"}]}`)
		case r.URL.EscapedPath() == "/view/Builds/config.xml" && r.Method == http.MethodGet:
			fmt.Fprint(w, listViewConfigFixture)
		case r.URL.EscapedPath() == "/view/Builds/configSubmit" && r.Method == http.MethodPost:
			if err := r.ParseForm(); err != nil {
				t.Errorf("ParseForm() error = %v", err)
			}
			submitted = r.PostForm
			w.WriteHeader(http.StatusFound)
		case r.URL.EscapedPath() == "/view/Builds2/config.xml":
			if r.Method == http.MethodGet {
				newPathGet++
				fmt.Fprint(w, listViewConfigFixture)
				return
			}
			body, _ := io.ReadAll(r.Body)
			finalPath = r.URL.EscapedPath()
			finalBody = string(body)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	name, err := client.UpdateView(context.Background(), "Builds", ViewUpdate{
		Name:           "Builds2",
		SetName:        true,
		Description:    "renamed",
		SetDescription: true,
	})
	if err != nil {
		t.Fatalf("UpdateView() error = %v", err)
	}
	if name != "Builds2" {
		t.Errorf("name = %q, want Builds2", name)
	}
	if submitted == nil {
		t.Fatal("configSubmit was not called")
	}
	if finalPath != "/view/Builds2/config.xml" {
		t.Errorf("final write path = %q", finalPath)
	}
	if !strings.Contains(finalBody, "<description>renamed</description>") {
		t.Errorf("description not patched: %s", finalBody)
	}
	if !strings.Contains(finalBody, "<string>demo</string>") {
		t.Errorf("job names dropped: %s", finalBody)
	}
	if newPathGet == 0 {
		t.Error("patched against cached XML instead of reading the renamed view")
	}
}

func TestUpdateViewRenameOnlyWritesOnce(t *testing.T) {
	var writes atomic.Int64
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/json" && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"views":[{"name":"Builds"}]}`)
		case r.URL.EscapedPath() == "/view/Builds/config.xml" && r.Method == http.MethodGet:
			fmt.Fprint(w, listViewConfigFixture)
		case r.URL.EscapedPath() == "/view/Builds/configSubmit" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusFound)
		case r.URL.EscapedPath() == "/view/Builds2/config.xml" && r.Method == http.MethodPost:
			writes.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	if _, err := client.UpdateView(context.Background(), "Builds", ViewUpdate{
		Name:    "Builds2",
		SetName: true,
	}); err != nil {
		t.Fatalf("UpdateView() error = %v", err)
	}
	if writes.Load() != 1 {
		t.Errorf("config.xml writes = %d, want 1", writes.Load())
	}
}

// TestRenameViewReportsPartialFailure 覆盖两步改名的中间态：
// configSubmit 成功但 XML 回写失败时，错误必须显式说明「名称已改」，
// 否则用户会以为什么都没发生，而 Jenkins 上视图已经换了名字、配置被简化。
func TestRenameViewReportsPartialFailure(t *testing.T) {
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/json" && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"views":[{"name":"Builds"}]}`)
		case r.URL.EscapedPath() == "/view/Builds/config.xml" && r.Method == http.MethodGet:
			fmt.Fprint(w, listViewConfigFixture)
		case r.URL.EscapedPath() == "/view/Builds/configSubmit" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusFound)
		case r.URL.EscapedPath() == "/view/Builds2/config.xml" && r.Method == http.MethodPost:
			http.Error(w, "boom", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	err := client.RenameView(context.Background(), "Builds", "Builds2")
	if err == nil {
		t.Fatal("RenameView() error = nil, want partial failure")
	}
	message := err.Error()
	if !strings.Contains(message, "Builds2") {
		t.Errorf("message = %q, want the new name", message)
	}
	if !strings.Contains(message, "restored") {
		t.Errorf("message = %q, want a hint that the config restore failed", message)
	}
}

func TestUpdateViewSameNameSkipsRename(t *testing.T) {
	var posts atomic.Int64
	server := newConfigServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.EscapedPath() == "/view/Builds/config.xml" && r.Method == http.MethodGet:
			fmt.Fprint(w, listViewConfigFixture)
		case r.URL.EscapedPath() == "/view/Builds/config.xml" && r.Method == http.MethodPost:
			posts.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	client := newTestClient(t, server.URL, false)
	name, err := client.UpdateView(context.Background(), "Builds", ViewUpdate{
		Name:           "Builds",
		SetName:        true,
		Description:    "new description",
		SetDescription: true,
	})
	if err != nil {
		t.Fatalf("UpdateView() error = %v", err)
	}
	if name != "Builds" {
		t.Errorf("name = %q, want Builds", name)
	}
	if posts.Load() != 1 {
		t.Errorf("config.xml writes = %d, want 1 (no rename)", posts.Load())
	}
}
