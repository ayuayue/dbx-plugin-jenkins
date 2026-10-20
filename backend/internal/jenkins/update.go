package jenkins

import (
	"context"
	"encoding/xml"
	"strings"
)

// JobFields 是任务编辑表单需要的可读字段。
type JobFields struct {
	Title       string `json:"title"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Disabled    bool   `json:"disabled"`
	Script      string `json:"script"`
}

// JobUpdate 描述一次任务编辑；Set* 用于区分「未修改」与「改成空值」。
type JobUpdate struct {
	Title          string
	SetTitle       bool
	Description    string
	SetDescription bool
	Disabled       bool
	SetDisabled    bool
	Script         string
	SetScript      bool
}

// ViewUpdate 描述一次视图编辑。
type ViewUpdate struct {
	Description     string
	SetDescription  bool
	JobNames        []string
	SetJobNames     bool
	IncludeRegex    string
	SetIncludeRegex bool
	Recurse         bool
	SetRecurse      bool
}

type jobConfigDocument struct {
	XMLName     xml.Name
	Description string `xml:"description"`
	Disabled    bool   `xml:"disabled"`
	Definition  *struct {
		Class  string `xml:"class,attr"`
		Script string `xml:"script"`
	} `xml:"definition"`
}

type listViewDocument struct {
	XMLName     xml.Name
	Name        string `xml:"name"`
	Description string `xml:"description"`
	JobNames    struct {
		Strings []string `xml:"string"`
	} `xml:"jobNames"`
	IncludeRegex string `xml:"includeRegex"`
	Recurse      bool   `xml:"recurse"`
}

// ParseJobFields 从 config.xml 读取编辑表单需要的字段。
func ParseJobFields(xmlText, fullName string) (*JobFields, error) {
	var document jobConfigDocument
	if err := xml.Unmarshal([]byte(normalizeXMLVersion(xmlText)), &document); err != nil {
		return nil, invalidXMLError(err.Error())
	}

	fields := &JobFields{
		Title:       lastSegment(fullName),
		Kind:        string(JobKindFreeStyle),
		Description: document.Description,
		Disabled:    document.Disabled,
	}
	if document.XMLName.Local == "flow-definition" {
		fields.Kind = string(JobKindPipeline)
	}
	if document.Definition != nil {
		fields.Script = document.Definition.Script
	}
	return fields, nil
}

// ParseListViewFields 从 config.xml 读取 ListView 的可编辑字段。
func ParseListViewFields(xmlText string) (ListViewConfig, error) {
	var document listViewDocument
	if err := xml.Unmarshal([]byte(normalizeXMLVersion(xmlText)), &document); err != nil {
		return ListViewConfig{}, invalidXMLError(err.Error())
	}
	if document.XMLName.Local != listViewClass && document.XMLName.Local != "listView" {
		return ListViewConfig{}, &APIError{
			Kind:    ErrInvalidConfig,
			Message: "view is not a ListView",
		}
	}
	return ListViewConfig{
		Name:         document.Name,
		Description:  document.Description,
		JobNames:     document.JobNames.Strings,
		IncludeRegex: document.IncludeRegex,
		Recurse:      document.Recurse,
	}, nil
}

// UpdateJob 按字段增量更新任务：先改名，再读原文、局部改写后写回。
func (c *Client) UpdateJob(ctx context.Context, fullName string, update JobUpdate) (string, error) {
	if c.readOnly {
		return "", &APIError{Kind: ErrReadOnly, Message: "connection is in read-only mode"}
	}

	current := strings.Trim(strings.TrimSpace(fullName), "/")
	if update.SetTitle {
		title := strings.TrimSpace(update.Title)
		if title != "" && title != lastSegment(current) {
			renamed, err := c.RenameJob(ctx, current, title)
			if err != nil {
				return "", err
			}
			current = renamed
		}
	}

	if !update.SetDescription && !update.SetDisabled && !update.SetScript {
		return current, nil
	}

	xmlText, err := c.GetJobConfig(ctx, current)
	if err != nil {
		return "", err
	}
	if update.SetDescription {
		if xmlText, err = PatchDescription(xmlText, update.Description); err != nil {
			return "", err
		}
	}
	if update.SetDisabled {
		if xmlText, err = PatchDisabled(xmlText, update.Disabled); err != nil {
			return "", err
		}
	}
	if update.SetScript {
		if xmlText, err = PatchPipelineScript(xmlText, update.Script); err != nil {
			return "", err
		}
	}
	if err := c.SaveJobConfig(ctx, current, xmlText); err != nil {
		return "", err
	}
	return current, nil
}

// UpdateView 按字段增量更新 ListView：读原文、合并未修改字段、局部改写后写回。
func (c *Client) UpdateView(ctx context.Context, name string, update ViewUpdate) error {
	if c.readOnly {
		return &APIError{Kind: ErrReadOnly, Message: "connection is in read-only mode"}
	}
	if !update.SetDescription && !update.SetJobNames &&
		!update.SetIncludeRegex && !update.SetRecurse {
		return nil
	}

	xmlText, err := c.GetViewConfig(ctx, name)
	if err != nil {
		return err
	}
	current, err := ParseListViewFields(xmlText)
	if err != nil {
		return err
	}
	if update.SetDescription {
		current.Description = update.Description
	}
	if update.SetJobNames {
		current.JobNames = update.JobNames
	}
	if update.SetIncludeRegex {
		current.IncludeRegex = update.IncludeRegex
	}
	if update.SetRecurse {
		current.Recurse = update.Recurse
	}

	patched, err := PatchListView(xmlText, current)
	if err != nil {
		return err
	}
	return c.SaveViewConfig(ctx, name, patched)
}

func lastSegment(fullName string) string {
	trimmed := strings.Trim(strings.TrimSpace(fullName), "/")
	if index := strings.LastIndex(trimmed, "/"); index >= 0 {
		return trimmed[index+1:]
	}
	return trimmed
}
