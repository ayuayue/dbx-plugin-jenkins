package jenkins

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

// JobKind 是新建任务时支持的类型。
type JobKind string

const (
	JobKindFreeStyle JobKind = "freestyle"
	JobKindPipeline  JobKind = "pipeline"
)

const (
	freeStyleClass = "hudson.model.FreeStyleProject"
	pipelineClass  = "org.jenkinsci.plugins.workflow.job.WorkflowJob"
	listViewClass  = "hudson.model.ListView"
)

// ListViewConfig 是 ListView 的可编辑字段。
type ListViewConfig struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	JobNames     []string `json:"jobNames"`
	IncludeRegex string   `json:"includeRegex"`
	Recurse      bool     `json:"recurse"`
}

// unsafeNameChars 是 Jenkins NameValidator 拒绝的字符（2.586 实测）。
// 其中 " 会让 Jenkins 返回 500，其余返回 400 "is an unsafe character"。
const unsafeNameChars = `/\?#%*:<>|&;!@$^[]"`

// ValidateName 校验任务/视图名称：Jenkins 把它们当作路径段和目录名。
func ValidateName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return &APIError{Kind: ErrInvalidConfig, Message: "name is required"}
	}
	if utf8.RuneCountInString(trimmed) > 255 {
		return &APIError{
			Kind:    ErrInvalidConfig,
			Message: "name must be 255 characters or fewer",
		}
	}
	if strings.Contains(trimmed, "..") {
		return &APIError{Kind: ErrInvalidConfig, Message: `name must not contain ".."`}
	}
	for _, char := range trimmed {
		if strings.ContainsRune(unsafeNameChars, char) {
			return &APIError{
				Kind:    ErrInvalidConfig,
				Message: fmt.Sprintf("name must not contain %q", string(char)),
			}
		}
	}
	for _, char := range trimmed {
		if char < 0x20 || char == 0x7f {
			return &APIError{Kind: ErrInvalidConfig, Message: "name must not contain control characters"}
		}
	}
	return nil
}

// JobConfigPath 返回任务的 config.xml 路径。
func JobConfigPath(fullName string) (string, error) {
	jobPath, err := JobPath(fullName)
	if err != nil {
		return "", &APIError{Kind: ErrInvalidConfig, Message: err.Error()}
	}
	return jobPath + "/config.xml", nil
}

// ViewConfigPath 返回视图的 config.xml 路径。
func ViewConfigPath(name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	return "/view/" + url.PathEscape(strings.TrimSpace(name)) + "/config.xml", nil
}

// GetJobConfig 读取任务配置原文。
func (c *Client) GetJobConfig(ctx context.Context, fullName string) (string, error) {
	path, err := JobConfigPath(fullName)
	if err != nil {
		return "", err
	}
	data, err := c.GetRaw(ctx, path, "application/xml")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveJobConfig 用 XML 原文覆盖任务配置。
func (c *Client) SaveJobConfig(ctx context.Context, fullName, xmlText string) error {
	if strings.TrimSpace(xmlText) == "" {
		return &APIError{Kind: ErrInvalidConfig, Message: "job config XML is required"}
	}
	path, err := JobConfigPath(fullName)
	if err != nil {
		return err
	}
	_, err = c.PostBody(ctx, path, "text/xml; charset=utf-8", []byte(xmlText))
	return err
}

// CreateJob 在指定文件夹下新建任务，返回任务 fullName。
func (c *Client) CreateJob(
	ctx context.Context,
	folder, name string,
	kind JobKind,
	description, script string,
) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}

	var xmlText string
	switch kind {
	case JobKindFreeStyle:
		xmlText = freeStyleConfigXML(description)
	case JobKindPipeline:
		xmlText = pipelineConfigXML(description, script)
	default:
		return "", &APIError{
			Kind:    ErrInvalidConfig,
			Message: fmt.Sprintf("unsupported job kind %q", kind),
		}
	}

	trimmedFolder := strings.Trim(strings.TrimSpace(folder), "/")
	base := "/"
	if trimmedFolder != "" {
		folderPath, err := JobPath(trimmedFolder)
		if err != nil {
			return "", &APIError{Kind: ErrInvalidConfig, Message: err.Error()}
		}
		base = folderPath + "/"
	}

	shortName := strings.TrimSpace(name)
	path := base + "createItem?name=" + url.QueryEscape(shortName)
	if _, err := c.PostBody(ctx, path, "text/xml; charset=utf-8", []byte(xmlText)); err != nil {
		return "", err
	}

	if trimmedFolder == "" {
		return shortName, nil
	}
	return trimmedFolder + "/" + shortName, nil
}

// RenameJob 重命名任务，返回新的 fullName。
func (c *Client) RenameJob(ctx context.Context, fullName, newName string) (string, error) {
	if err := ValidateName(newName); err != nil {
		return "", err
	}
	jobPath, err := JobPath(fullName)
	if err != nil {
		return "", &APIError{Kind: ErrInvalidConfig, Message: err.Error()}
	}

	shortName := strings.TrimSpace(newName)
	path := jobPath + "/doRename?newName=" + url.QueryEscape(shortName)
	if _, err := c.PostBody(ctx, path, "application/x-www-form-urlencoded", nil); err != nil {
		return "", err
	}

	trimmed := strings.Trim(strings.TrimSpace(fullName), "/")
	if index := strings.LastIndex(trimmed, "/"); index >= 0 {
		return trimmed[:index] + "/" + shortName, nil
	}
	return shortName, nil
}

// DeleteJob 删除任务。
func (c *Client) DeleteJob(ctx context.Context, fullName string) error {
	jobPath, err := JobPath(fullName)
	if err != nil {
		return &APIError{Kind: ErrInvalidConfig, Message: err.Error()}
	}
	_, err = c.PostBody(ctx, jobPath+"/doDelete", "application/x-www-form-urlencoded", nil)
	return err
}

// GetViewConfig 读取视图配置原文。
func (c *Client) GetViewConfig(ctx context.Context, name string) (string, error) {
	path, err := ViewConfigPath(name)
	if err != nil {
		return "", err
	}
	data, err := c.GetRaw(ctx, path, "application/xml")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveViewConfig 用 XML 原文覆盖视图配置。
func (c *Client) SaveViewConfig(ctx context.Context, name, xmlText string) error {
	if strings.TrimSpace(xmlText) == "" {
		return &APIError{Kind: ErrInvalidConfig, Message: "view config XML is required"}
	}
	path, err := ViewConfigPath(name)
	if err != nil {
		return err
	}
	_, err = c.PostBody(ctx, path, "text/xml; charset=utf-8", []byte(xmlText))
	return err
}

// CreateView 新建 ListView。
func (c *Client) CreateView(ctx context.Context, cfg ListViewConfig) error {
	name := strings.TrimSpace(cfg.Name)
	if err := ValidateName(name); err != nil {
		return err
	}

	jobNames := cfg.JobNames
	if jobNames == nil {
		jobNames = []string{}
	}
	payload := map[string]any{
		"name":            name,
		"mode":            listViewClass,
		"description":     cfg.Description,
		"jobNames":        jobNames,
		"recurse":         cfg.Recurse,
		"filterExecutors": false,
		"filterQueue":     false,
		"columns":         listViewColumns(),
	}
	if cfg.IncludeRegex != "" {
		payload["includeRegex"] = cfg.IncludeRegex
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return &APIError{Kind: ErrRequest, Message: "cannot encode view configuration"}
	}

	// Jenkins 的 /createView 要求 name 与 mode 同时出现在表单和 json 里，否则 400。
	form := url.Values{}
	form.Set("name", name)
	form.Set("mode", listViewClass)
	form.Set("json", string(encoded))
	_, err = c.PostForm(ctx, "/createView", form)
	return err
}

// DeleteView 删除视图。
func (c *Client) DeleteView(ctx context.Context, name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	path := "/view/" + url.PathEscape(strings.TrimSpace(name)) + "/doDelete"
	_, err := c.PostBody(ctx, path, "application/x-www-form-urlencoded", nil)
	return err
}

// listViewColumns 复刻 Jenkins 新建 ListView 的默认列，避免视图变成空白列表。
func listViewColumns() []map[string]string {
	classes := []string{
		"hudson.views.StatusColumn",
		"hudson.views.WeatherColumn",
		"hudson.views.JobColumn",
		"hudson.views.LastSuccessColumn",
		"hudson.views.LastFailureColumn",
		"hudson.views.LastDurationColumn",
		"hudson.views.BuildButtonColumn",
	}
	columns := make([]map[string]string, 0, len(classes))
	for _, class := range classes {
		columns = append(columns, map[string]string{
			"stapler-class": class,
			"kind":          class,
		})
	}
	return columns
}

func freeStyleConfigXML(description string) string {
	return `<?xml version='1.1' encoding='UTF-8'?>
<project>
  <description>` + escapeXMLText(description) + `</description>
  <keepDependencies>false</keepDependencies>
  <properties/>
  <scm class="hudson.scm.NullSCM"/>
  <canRoam>true</canRoam>
  <disabled>false</disabled>
  <blockBuildWhenDownstreamBuilding>false</blockBuildWhenDownstreamBuilding>
  <blockBuildWhenUpstreamBuilding>false</blockBuildWhenUpstreamBuilding>
  <triggers/>
  <concurrentBuild>false</concurrentBuild>
  <builders/>
  <publishers/>
  <buildWrappers/>
</project>`
}

func pipelineConfigXML(description, script string) string {
	return `<?xml version='1.1' encoding='UTF-8'?>
<flow-definition>
  <description>` + escapeXMLText(description) + `</description>
  <keepDependencies>false</keepDependencies>
  <properties/>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition">
    <script>` + escapeXMLText(script) + `</script>
    <sandbox>true</sandbox>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>`
}

// escapeXMLText 只转义文本节点必需的三个字符，保留引号和换行。
func escapeXMLText(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))
	for _, char := range text {
		switch char {
		case '&':
			builder.WriteString("&amp;")
		case '<':
			builder.WriteString("&lt;")
		case '>':
			builder.WriteString("&gt;")
		default:
			builder.WriteRune(char)
		}
	}
	return builder.String()
}
