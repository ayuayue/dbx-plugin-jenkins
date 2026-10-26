package jenkins

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

const (
	// 批量读作业信息的默认/最大并发：读写分离，读可以并发，写一律串行。
	defaultBatchConcurrency = 4
	maxBatchConcurrency     = 8
)

// 批量读/预检共用的字段树：状态、队列、参数定义（含类型与可选值，供批量参数表单使用）。
// 不用 jobTree 全量树，避免为 N 个任务拉回 description / lastBuild 等无用字段。
const batchJobTree = "name,fullName,buildable,inQueue,color," +
	"property[parameterDefinitions[name,type,description,defaultParameterValue[value],choices]]"

// BatchBuildOptions 控制批量触发行为。
type BatchBuildOptions struct {
	// SkipBusy 为真时跳过「已在队列」与「正在构建」的任务。调用方（RPC 层）负责给默认值 true。
	SkipBusy bool
	// MaxConcurrency 是读作业信息的并发上限，0 走默认值，超出上限按上限截断。
	MaxConcurrency int
	// Params 是本批共用参数。只应用到「该任务自己声明过」的参数名上；
	// 其余任务的未声明参数不会提交，避免 Jenkins 收到未知参数。
	Params url.Values
}

// BatchBuildResult 是单个任务的触发结果，切片顺序与入参一致。
type BatchBuildResult struct {
	Job     string `json:"job"`
	Success bool   `json:"success"`
	Skipped bool   `json:"skipped"`
	QueueID int    `json:"queueId"`
	// Reason 是跳过/失败原因（前端据此本地化）：
	// not_buildable | in_queue | building | 以及 ErrorKind 的取值。
	Reason string `json:"reason"`
	Error  string `json:"error"`
}

// batchJobInfo 是批量触发前的任务快照。
type batchJobInfo struct {
	Name      string `json:"name"`
	FullName  string `json:"fullName"`
	Buildable bool   `json:"buildable"`
	InQueue   bool   `json:"inQueue"`
	Color     string `json:"color"`
	Property  []struct {
		ParameterDefinitions []struct {
			Name                  string   `json:"name"`
			Type                  string   `json:"type"`
			Description           string   `json:"description"`
			Choices               []string `json:"choices"`
			DefaultParameterValue *struct {
				Value any `json:"value"`
			} `json:"defaultParameterValue"`
		} `json:"parameterDefinitions"`
	} `json:"property"`
}

// parameters 把 Jenkins 的嵌套参数定义摊平成统一模型（默认值转字符串）。
func (info *batchJobInfo) parameters() []ParameterDefinition {
	parameters := make([]ParameterDefinition, 0, 4)
	for _, property := range info.Property {
		for _, definition := range property.ParameterDefinitions {
			name := strings.TrimSpace(definition.Name)
			if name == "" {
				continue
			}
			parameter := ParameterDefinition{
				Name:        name,
				Type:        definition.Type,
				Description: definition.Description,
				Choices:     definition.Choices,
			}
			if definition.DefaultParameterValue != nil {
				parameter.DefaultValue = anyToString(definition.DefaultParameterValue.Value)
			}
			parameters = append(parameters, parameter)
		}
	}
	return parameters
}

// defaults 把参数默认值转成 buildWithParameters 的表单。
// 参数化任务必须带参走带参端点，缺参在部分 Jenkins 版本里会让构建读到 null，所以缺省值也提交空串。
func (info *batchJobInfo) defaults() url.Values {
	form := url.Values{}
	for _, parameter := range info.parameters() {
		form.Set(parameter.Name, parameter.DefaultValue)
	}
	return form
}

// buildForm 在默认值基础上叠加共享参数：只覆盖该任务声明过的参数名。
// 直接赋值整个切片而不是 Set，保留多选参数的多值语义。
func (info *batchJobInfo) buildForm(shared url.Values) url.Values {
	form := info.defaults()
	if len(shared) == 0 {
		return form
	}
	for _, parameter := range info.parameters() {
		if values, ok := shared[parameter.Name]; ok {
			form[parameter.Name] = values
		}
	}
	return form
}

// target 转成预检结果（确认弹窗用的实时状态 + 参数定义）。
func (info *batchJobInfo) target(fallback string) BatchTargetInfo {
	fullName := strings.TrimSpace(info.FullName)
	if fullName == "" {
		fullName = fallback
	}
	name := strings.TrimSpace(info.Name)
	if name == "" {
		parts := strings.Split(fullName, "/")
		name = parts[len(parts)-1]
	}
	return BatchTargetInfo{
		Name:       name,
		FullName:   fullName,
		Status:     NormalizeColor(info.Color),
		Building:   isBuildingColor(info.Color),
		Buildable:  info.Buildable,
		InQueue:    info.InQueue,
		Parameters: info.parameters(),
	}
}

// BatchBuild 批量触发构建：并发读任务信息，串行触发，逐条返回结果。
//
// 单条任务失败不会中断其余任务；只读连接直接失败且不发任何请求。
// 这里不做去重，调用方需保证任务名唯一。
func (c *Client) BatchBuild(
	ctx context.Context,
	names []string,
	opts BatchBuildOptions,
) ([]BatchBuildResult, error) {
	if len(names) == 0 {
		return []BatchBuildResult{}, nil
	}
	if c.ReadOnly() {
		return nil, &APIError{Kind: ErrReadOnly, Message: "connection is in read-only mode"}
	}

	concurrency := batchConcurrency(opts.MaxConcurrency)

	results := make([]BatchBuildResult, len(names))
	infos := make([]*batchJobInfo, len(names))
	readErrs := make([]error, len(names))

	// 读阶段并发：拉参数定义是纯读，互不影响。
	sem := make(chan struct{}, concurrency)
	var wait sync.WaitGroup
	for index := range names {
		wait.Add(1)
		go func(slot int) {
			defer wait.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			info, err := c.getBatchJobInfo(ctx, names[slot])
			if err != nil {
				readErrs[slot] = err
				return
			}
			infos[slot] = info
		}(index)
	}
	wait.Wait()

	// 写阶段串行：CSRF crumb 没有 singleflight，并发首写会互相踩；
	// 串行还能让 Jenkins 队列序号与任务顺序对齐。
	for index, name := range names {
		if readErrs[index] != nil {
			results[index] = batchFailure(name, readErrs[index])
			continue
		}
		info := infos[index]
		if info == nil {
			results[index] = BatchBuildResult{Job: name, Reason: "request_failed", Error: "job info missing"}
			continue
		}
		if !info.Buildable {
			results[index] = BatchBuildResult{Job: name, Skipped: true, Reason: "not_buildable"}
			continue
		}
		if opts.SkipBusy && info.InQueue {
			results[index] = BatchBuildResult{Job: name, Skipped: true, Reason: "in_queue"}
			continue
		}
		if opts.SkipBusy && isBuildingColor(info.Color) {
			results[index] = BatchBuildResult{Job: name, Skipped: true, Reason: "building"}
			continue
		}

		queueID, err := c.TriggerBuild(ctx, name, info.buildForm(opts.Params))
		if err != nil {
			results[index] = batchFailure(name, err)
			continue
		}
		results[index] = BatchBuildResult{Job: name, Success: true, QueueID: queueID}
	}
	return results, nil
}

// InspectBatch 读取批量预检信息：实时状态与参数定义。
//
// 单个任务读取失败不会让整个预检失败，该条带 Error 返回（前端仍可展示其余任务）；
// 顺序与入参一致。
func (c *Client) InspectBatch(
	ctx context.Context,
	names []string,
	concurrency int,
) ([]BatchTargetInfo, error) {
	if len(names) == 0 {
		return []BatchTargetInfo{}, nil
	}

	width := batchConcurrency(concurrency)
	targets := make([]BatchTargetInfo, len(names))
	sem := make(chan struct{}, width)
	var wait sync.WaitGroup
	for index := range names {
		wait.Add(1)
		go func(slot int) {
			defer wait.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			name := names[slot]
			info, err := c.getBatchJobInfo(ctx, name)
			if err != nil {
				targets[slot] = BatchTargetInfo{
					Name:     name,
					FullName: name,
					Status:   StatusUnknown,
					Error:    err.Error(),
				}
				return
			}
			targets[slot] = info.target(name)
		}(index)
	}
	wait.Wait()
	return targets, nil
}

func batchConcurrency(value int) int {
	if value <= 0 {
		return defaultBatchConcurrency
	}
	if value > maxBatchConcurrency {
		return maxBatchConcurrency
	}
	return value
}

// getBatchJobInfo 读取批量触发需要的任务快照。
func (c *Client) getBatchJobInfo(ctx context.Context, fullName string) (*batchJobInfo, error) {
	jobPath, err := JobPath(fullName)
	if err != nil {
		return nil, &APIError{Kind: ErrInvalidConfig, Message: err.Error()}
	}
	query := url.Values{"tree": {batchJobTree}}
	var info batchJobInfo
	if err := c.Get(ctx, jobPath+"/api/json?"+query.Encode(), &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// isBuildingColor 判定任务颜色是否处于构建中（Jenkins 用 *_anime 后缀表示）。
func isBuildingColor(color string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(color)), "_anime")
}

// batchFailure 把错误归一化成单条失败结果。
func batchFailure(name string, err error) BatchBuildResult {
	result := BatchBuildResult{Job: name, Reason: "request_failed", Error: err.Error()}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		result.Reason = string(apiErr.Kind)
		result.Error = apiErr.Message
		if apiErr.StatusCode > 0 {
			result.Error = apiErr.Message + " (HTTP " + strconv.Itoa(apiErr.StatusCode) + ")"
		}
	}
	return result
}
