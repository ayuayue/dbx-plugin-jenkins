package jenkins

import (
	"context"
	"net/url"
	"strings"
)

// GetViews 读取 Jenkins 的视图列表。
func (c *Client) GetViews(ctx context.Context) ([]View, error) {
	query := url.Values{"tree": {"views[name,description,url]"}}
	var payload struct {
		Views []View `json:"views"`
	}
	if err := c.Get(ctx, "/api/json?"+query.Encode(), &payload); err != nil {
		return nil, err
	}
	return payload.Views, nil
}

// GetViewJobs 读取指定视图内的任务。
func (c *Client) GetViewJobs(ctx context.Context, view string) ([]JobSummary, error) {
	name := strings.TrimSpace(view)
	if name == "" {
		return nil, &APIError{Kind: ErrInvalidConfig, Message: "view name is required"}
	}

	path := "/view/" + url.PathEscape(name) + "/api/json"
	query := url.Values{"tree": {jobsTree}}
	var payload struct {
		Jobs []JobSummary `json:"jobs"`
	}
	if err := c.Get(ctx, path+"?"+query.Encode(), &payload); err != nil {
		return nil, err
	}
	return normalizeJobs(payload.Jobs, ""), nil
}
