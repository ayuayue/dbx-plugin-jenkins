package jenkins

import (
	"errors"
	"strings"
	"testing"
)

const jobConfigFixture = `<?xml version='1.1' encoding='UTF-8'?>
<project>
  <description>old description</description>
  <keepDependencies>false</keepDependencies>
  <scm class="hudson.plugins.git.GitSCM">
    <userRemoteConfigs>
      <hudson.plugins.git.UserRemoteConfig>
        <url>https://example.com/repo.git</url>
      </hudson.plugins.git.UserRemoteConfig>
    </userRemoteConfigs>
  </scm>
  <builders>
    <hudson.tasks.Shell>
      <command>echo hi</command>
    </hudson.tasks.Shell>
  </builders>
  <disabled>false</disabled>
</project>`

const pipelineConfigFixture = `<?xml version='1.1' encoding='UTF-8'?>
<flow-definition>
  <description>nightly</description>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition">
    <script>echo old</script>
    <sandbox>true</sandbox>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>`

const listViewConfigFixture = `<?xml version='1.1' encoding='UTF-8'?>
<hudson.model.ListView>
  <name>Builds</name>
  <description>old description</description>
  <filterExecutors>false</filterExecutors>
  <filterQueue>false</filterQueue>
  <properties class="hudson.model.View$PropertyList"/>
  <jobNames>
    <comparator class="hudson.util.CaseInsensitiveComparator"/>
    <string>demo</string>
  </jobNames>
  <jobFilters/>
  <columns>
    <hudson.views.StatusColumn/>
    <hudson.views.JobColumn/>
  </columns>
  <recurse>false</recurse>
</hudson.model.ListView>`

func TestPatchDescriptionReplacesAndPreserves(t *testing.T) {
	patched, err := PatchDescription(jobConfigFixture, "new & <safe>")
	if err != nil {
		t.Fatalf("PatchDescription() error = %v", err)
	}
	if !strings.Contains(patched, "<description>new &amp; &lt;safe&gt;</description>") {
		t.Errorf("description not replaced/escaped:\n%s", patched)
	}
	if strings.Contains(patched, "old description") {
		t.Errorf("old description still present:\n%s", patched)
	}
	if !strings.Contains(patched, "hudson.plugins.git.GitSCM") ||
		!strings.Contains(patched, "https://example.com/repo.git") {
		t.Errorf("SCM config was dropped:\n%s", patched)
	}
	if !strings.Contains(patched, "<hudson.tasks.Shell>") ||
		!strings.Contains(patched, "<command>echo hi</command>") {
		t.Errorf("builders were dropped:\n%s", patched)
	}
	if !strings.Contains(patched, "<keepDependencies>false</keepDependencies>") ||
		!strings.Contains(patched, "<disabled>false</disabled>") {
		t.Errorf("sibling elements were dropped:\n%s", patched)
	}
}

func TestPatchDescriptionInsertsWhenMissing(t *testing.T) {
	patched, err := PatchDescription("<project><disabled>false</disabled></project>", "hello")
	if err != nil {
		t.Fatalf("PatchDescription() error = %v", err)
	}
	if !strings.Contains(patched, "<description>hello</description>") {
		t.Errorf("description not inserted:\n%s", patched)
	}
	if !strings.Contains(patched, "<disabled>false</disabled>") {
		t.Errorf("siblings dropped:\n%s", patched)
	}
}

func TestPatchDisabled(t *testing.T) {
	patched, err := PatchDisabled(jobConfigFixture, true)
	if err != nil {
		t.Fatalf("PatchDisabled() error = %v", err)
	}
	if !strings.Contains(patched, "<disabled>true</disabled>") {
		t.Errorf("disabled not set:\n%s", patched)
	}

	patched, err = PatchDisabled("<project><description>x</description></project>", true)
	if err != nil {
		t.Fatalf("PatchDisabled(insert) error = %v", err)
	}
	if !strings.Contains(patched, "<disabled>true</disabled>") {
		t.Errorf("disabled not inserted:\n%s", patched)
	}
}

func TestPatchPipelineScript(t *testing.T) {
	patched, err := PatchPipelineScript(pipelineConfigFixture, "echo new\nsh 'ls -l'")
	if err != nil {
		t.Fatalf("PatchPipelineScript() error = %v", err)
	}
	if !strings.Contains(patched, "echo new") || !strings.Contains(patched, "sh 'ls -l'") {
		t.Errorf("script not replaced:\n%s", patched)
	}
	if strings.Contains(patched, "echo old") {
		t.Errorf("old script still present:\n%s", patched)
	}
	if !strings.Contains(patched, "<sandbox>true</sandbox>") {
		t.Errorf("sandbox dropped:\n%s", patched)
	}
	if !strings.Contains(patched, "<description>nightly</description>") {
		t.Errorf("description dropped:\n%s", patched)
	}
}

func TestPatchPipelineScriptEscapesXML(t *testing.T) {
	patched, err := PatchPipelineScript(pipelineConfigFixture, `echo "a < b && c > d"`)
	if err != nil {
		t.Fatalf("PatchPipelineScript() error = %v", err)
	}
	if !strings.Contains(patched, "a &lt; b &amp;&amp; c &gt; d") {
		t.Errorf("script not escaped:\n%s", patched)
	}
}

func TestPatchPipelineScriptMissingDefinition(t *testing.T) {
	_, err := PatchPipelineScript(jobConfigFixture, "echo hi")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrInvalidConfig {
		t.Fatalf("error = %v, want invalid_config", err)
	}
}

func TestPatchListView(t *testing.T) {
	patched, err := PatchListView(listViewConfigFixture, ListViewConfig{
		Description:  "managed by dbx",
		JobNames:     []string{"api", "web"},
		IncludeRegex: "^release-.*",
		Recurse:      true,
	})
	if err != nil {
		t.Fatalf("PatchListView() error = %v", err)
	}

	if !strings.Contains(patched, "<description>managed by dbx</description>") {
		t.Errorf("description not replaced:\n%s", patched)
	}
	if !strings.Contains(patched, "<string>api</string>") ||
		!strings.Contains(patched, "<string>web</string>") {
		t.Errorf("job names not replaced:\n%s", patched)
	}
	if strings.Contains(patched, "<string>demo</string>") {
		t.Errorf("old job name still present:\n%s", patched)
	}
	if !strings.Contains(patched, `class="hudson.util.CaseInsensitiveComparator"`) {
		t.Errorf("comparator dropped:\n%s", patched)
	}
	if !strings.Contains(patched, "<includeRegex>^release-.*</includeRegex>") {
		t.Errorf("includeRegex not inserted:\n%s", patched)
	}
	if !strings.Contains(patched, "<recurse>true</recurse>") {
		t.Errorf("recurse not updated:\n%s", patched)
	}
	if !strings.Contains(patched, "<hudson.views.StatusColumn/>") ||
		!strings.Contains(patched, "<hudson.views.JobColumn/>") {
		t.Errorf("columns dropped:\n%s", patched)
	}
	if !strings.Contains(patched, "<filterExecutors>false</filterExecutors>") ||
		!strings.Contains(patched, "<properties class=\"hudson.model.View$PropertyList\"/>") {
		t.Errorf("sibling elements dropped:\n%s", patched)
	}
}

func TestPatchListViewRemovesEmptyIncludeRegex(t *testing.T) {
	withRegex := strings.Replace(
		listViewConfigFixture,
		"<recurse>false</recurse>",
		"<includeRegex>^old-.*</includeRegex>\n  <recurse>false</recurse>",
		1,
	)
	patched, err := PatchListView(withRegex, ListViewConfig{Description: "x"})
	if err != nil {
		t.Fatalf("PatchListView() error = %v", err)
	}
	if strings.Contains(patched, "includeRegex") {
		t.Errorf("includeRegex not removed:\n%s", patched)
	}
}

func TestPatchListViewRejectsNonListView(t *testing.T) {
	_, err := PatchListView(jobConfigFixture, ListViewConfig{Description: "x"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != ErrInvalidConfig {
		t.Fatalf("error = %v, want invalid_config", err)
	}
}

func TestPatchRejectsInvalidXML(t *testing.T) {
	if _, err := PatchDescription("<project><description>x</project>", "y"); err == nil {
		t.Fatal("PatchDescription() error = nil, want an error")
	}
	if _, err := PatchListView("<hudson.model.ListView><name>x</hudson.model.ListView>", ListViewConfig{}); err == nil {
		t.Fatal("PatchListView() error = nil, want an error")
	}
}
