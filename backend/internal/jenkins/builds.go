package jenkins

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// maxChangeMessageRunes 是变更信息截断上限，避免矩阵响应体过大。
const maxChangeMessageRunes = 200

// rawChange 是 Jenkins changelog 里的一条提交。
type rawChange struct {
	CommitID string `json:"commitId"`
	Message  string `json:"msg"`
	Author   struct {
		FullName string `json:"fullName"`
	} `json:"author"`
}

type rawChangeSet struct {
	Items []rawChange `json:"items"`
}

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
	// Freestyle 用 changeSet，Pipeline（WorkflowRun）的 changelog 只出现在 changeSets。
	ChangeSet  rawChangeSet   `json:"changeSet"`
	ChangeSets []rawChangeSet `json:"changeSets"`
}

func (raw rawBuild) causes() []Cause {
	var causes []Cause
	for _, action := range raw.Actions {
		causes = append(causes, action.Causes...)
	}
	return causes
}

// changes 把构建的变更集归一化为阶段矩阵模型；没有变更时返回空切片而不是 nil。
func (raw rawBuild) changes() []ChangeSetItem {
	sets := make([][]rawChange, 0, 1+len(raw.ChangeSets))
	sets = append(sets, raw.ChangeSet.Items)
	for _, set := range raw.ChangeSets {
		sets = append(sets, set.Items)
	}

	changes := mergeChanges(sets...)
	for index := range changes {
		changes[index].Message = truncateChangeMessage(changes[index].Message)
	}
	return changes
}

// mergeChanges 合并单数/复数 changelog，按 commitId 去重后归一化（保留完整信息）。
func mergeChanges(sets ...[]rawChange) []ChangeSetItem {
	total := 0
	for _, items := range sets {
		total += len(items)
	}

	changes := make([]ChangeSetItem, 0, total)
	seen := make(map[string]bool, total)
	for _, items := range sets {
		for _, item := range items {
			commitID := strings.TrimSpace(item.CommitID)
			if commitID != "" {
				if seen[commitID] {
					continue
				}
				seen[commitID] = true
			}
			changes = append(changes, ChangeSetItem{
				CommitID: commitID,
				Message:  strings.TrimSpace(item.Message),
				Author:   strings.TrimSpace(item.Author.FullName),
			})
		}
	}
	return changes
}

// truncateChangeMessage 去掉首尾空白；超长时按 rune 截断并补省略号，避免截断多字节字符。
func truncateChangeMessage(message string) string {
	message = strings.TrimSpace(message)
	runes := []rune(message)
	if len(runes) <= maxChangeMessageRunes {
		return message
	}
	return string(runes[:maxChangeMessageRunes]) + "…"
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
		"builds[number,result,building,timestamp,duration,displayName,actions[causes[shortDescription,userId,userName]],changeSet[items[commitId,msg,author[fullName]]],changeSets[items[commitId,msg,author[fullName]]]]{0,%d}",
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
		changes := raw.changes()
		builds = append(builds, BuildSummary{
			Number:      raw.Number,
			Result:      raw.Result,
			Building:    raw.Building,
			Timestamp:   raw.Timestamp,
			Duration:    raw.Duration,
			DisplayName: raw.DisplayName,
			Causes:      raw.causes(),
			Status:      StatusFromResult(raw.Result, raw.Building),
			CommitCount: len(changes),
			Changes:     changes,
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
		"changeSets[items[commitId,msg,author[fullName]]]," +
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
		ChangeSet  rawChangeSet   `json:"changeSet"`
		ChangeSets []rawChangeSet `json:"changeSets"`
		Artifacts  []Artifact     `json:"artifacts"`
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
	sets := make([][]rawChange, 0, 1+len(payload.ChangeSets))
	sets = append(sets, payload.ChangeSet.Items)
	for _, set := range payload.ChangeSets {
		sets = append(sets, set.Items)
	}
	build.ChangeSet = mergeChanges(sets...)
	return build, nil
}
