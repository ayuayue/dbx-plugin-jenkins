package jenkins

import (
	"context"
	"fmt"
	"net/url"
)

type rawBuild struct {
	Number      int    `json:"number"`
	Result      string `json:"result"`
	Building    bool   `json:"building"`
	Timestamp   int64  `json:"timestamp"`
	Duration    int64  `json:"duration"`
	DisplayName string `json:"displayName"`
	Actions     []struct {
		Causes []Cause `json:"causes"`
	} `json:"actions"`
	ChangeSet struct {
		Items []struct {
			CommitID string `json:"commitId"`
		} `json:"items"`
	} `json:"changeSet"`
}

func (raw rawBuild) causes() []Cause {
	var causes []Cause
	for _, action := range raw.Actions {
		causes = append(causes, action.Causes...)
	}
	return causes
}

// GetBuilds 读取任务最近的构建列表。
func (c *Client) GetBuilds(ctx context.Context, fullName string, limit int) ([]BuildSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	jobPath, err := JobPath(fullName)
	if err != nil {
		return nil, &APIError{Kind: ErrInvalidConfig, Message: err.Error()}
	}

	tree := fmt.Sprintf(
		"builds[number,result,building,timestamp,duration,displayName,actions[causes[shortDescription,userId,userName]],changeSet[items[commitId]]]{0,%d}",
		limit,
	)
	var payload struct {
		Builds []rawBuild `json:"builds"`
	}
	query := url.Values{"tree": {tree}}
	if err := c.Get(ctx, jobPath+"/api/json?"+query.Encode(), &payload); err != nil {
		return nil, err
	}

	builds := make([]BuildSummary, 0, len(payload.Builds))
	for _, raw := range payload.Builds {
		builds = append(builds, BuildSummary{
			Number:      raw.Number,
			Result:      raw.Result,
			Building:    raw.Building,
			Timestamp:   raw.Timestamp,
			Duration:    raw.Duration,
			DisplayName: raw.DisplayName,
			Causes:      raw.causes(),
			Status:      StatusFromResult(raw.Result, raw.Building),
			CommitCount: len(raw.ChangeSet.Items),
		})
	}
	return builds, nil
}

// GetBuild 读取单次构建详情。
func (c *Client) GetBuild(ctx context.Context, fullName string, number int) (*BuildDetail, error) {
	jobPath, err := JobPath(fullName)
	if err != nil {
		return nil, &APIError{Kind: ErrInvalidConfig, Message: err.Error()}
	}
	if number <= 0 {
		return nil, &APIError{Kind: ErrInvalidConfig, Message: "build number must be positive"}
	}

	tree := "number,result,building,timestamp,duration,estimatedDuration,displayName,fullDisplayName,url," +
		"actions[causes[shortDescription,userId,userName]]," +
		"changeSet[items[commitId,msg,author[fullName]]]," +
		"artifacts[fileName,relativePath]"
	var payload struct {
		Number            int    `json:"number"`
		Result            string `json:"result"`
		Building          bool   `json:"building"`
		Timestamp         int64  `json:"timestamp"`
		Duration          int64  `json:"duration"`
		EstimatedDuration int64  `json:"estimatedDuration"`
		DisplayName       string `json:"displayName"`
		FullDisplayName   string `json:"fullDisplayName"`
		URL               string `json:"url"`
		Actions           []struct {
			Causes []Cause `json:"causes"`
		} `json:"actions"`
		ChangeSet struct {
			Items []struct {
				CommitID string `json:"commitId"`
				Message  string `json:"msg"`
				Author   struct {
					FullName string `json:"fullName"`
				} `json:"author"`
			} `json:"items"`
		} `json:"changeSet"`
		Artifacts []Artifact `json:"artifacts"`
	}
	query := url.Values{"tree": {tree}}
	path := fmt.Sprintf("%s/%d/api/json?%s", jobPath, number, query.Encode())
	if err := c.Get(ctx, path, &payload); err != nil {
		return nil, err
	}

	build := &BuildDetail{
		Number:            payload.Number,
		Result:            payload.Result,
		Building:          payload.Building,
		Timestamp:         payload.Timestamp,
		Duration:          payload.Duration,
		EstimatedDuration: payload.EstimatedDuration,
		DisplayName:       payload.DisplayName,
		FullDisplayName:   payload.FullDisplayName,
		URL:               payload.URL,
		Artifacts:         payload.Artifacts,
		Status:            StatusFromResult(payload.Result, payload.Building),
	}
	for _, action := range payload.Actions {
		build.Causes = append(build.Causes, action.Causes...)
	}
	for _, item := range payload.ChangeSet.Items {
		build.ChangeSet = append(build.ChangeSet, ChangeSetItem{
			CommitID: item.CommitID,
			Message:  item.Message,
			Author:   item.Author.FullName,
		})
	}
	return build, nil
}
