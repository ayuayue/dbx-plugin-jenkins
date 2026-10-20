package jenkins

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

const maxStageBuilds = 20

// GetBuildStages 读取单次 Pipeline 构建的阶段。
func (c *Client) GetBuildStages(ctx context.Context, job string, number int) ([]Stage, error) {
	jobPath, err := JobPath(job)
	if err != nil {
		return nil, &APIError{Kind: ErrInvalidConfig, Message: err.Error()}
	}
	if number <= 0 {
		return nil, &APIError{Kind: ErrInvalidConfig, Message: "build number must be positive"}
	}

	path := fmt.Sprintf("%s/%d/wfapi/describe", jobPath, number)
	var payload struct {
		Stages []Stage `json:"stages"`
	}
	if err := c.Get(ctx, path, &payload); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Kind == ErrNotFound {
			return nil, &APIError{
				Kind:    ErrStagesUnavailable,
				Message: "pipeline stage data is not available (pipeline-rest-api returned 404)",
			}
		}
		return nil, err
	}

	for index := range payload.Stages {
		payload.Stages[index].Status = strings.ToLower(strings.TrimSpace(payload.Stages[index].Status))
	}
	return payload.Stages, nil
}

// GetStageMatrix 读取最近若干次构建的阶段矩阵。
func (c *Client) GetStageMatrix(ctx context.Context, job string, limit int) (*StageMatrix, error) {
	if strings.TrimSpace(job) == "" {
		return nil, &APIError{Kind: ErrInvalidConfig, Message: "job name is required"}
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > maxStageBuilds {
		limit = maxStageBuilds
	}

	builds, err := c.GetBuilds(ctx, job, limit)
	if err != nil {
		return nil, err
	}

	stages := make([][]Stage, len(builds))
	errs := make([]error, len(builds))
	sem := make(chan struct{}, 4)
	var wait sync.WaitGroup

	for index := range builds {
		wait.Add(1)
		sem <- struct{}{}
		go func(index int) {
			defer wait.Done()
			defer func() { <-sem }()
			stages[index], errs[index] = c.GetBuildStages(ctx, job, builds[index].Number)
		}(index)
	}
	wait.Wait()

	unavailable := 0
	for _, fetchErr := range errs {
		if fetchErr == nil {
			continue
		}
		var apiErr *APIError
		if errors.As(fetchErr, &apiErr) && apiErr.Kind == ErrStagesUnavailable {
			unavailable++
			continue
		}
		return nil, fetchErr
	}
	if len(builds) > 0 && unavailable == len(builds) {
		return nil, &APIError{
			Kind:    ErrStagesUnavailable,
			Message: "pipeline stage data is not available for the recent builds",
		}
	}

	matrix := &StageMatrix{
		Job:    job,
		Builds: make([]StageBuild, 0, len(builds)),
	}
	seen := make(map[string]bool)
	for index := range builds {
		for _, stage := range stages[index] {
			if stage.Name == "" || seen[stage.Name] {
				continue
			}
			seen[stage.Name] = true
			matrix.StageNames = append(matrix.StageNames, stage.Name)
		}
	}

	positions := make(map[string]int, len(matrix.StageNames))
	for index, name := range matrix.StageNames {
		positions[name] = index
	}
	sums := make([]int64, len(matrix.StageNames))
	counts := make([]int, len(matrix.StageNames))

	for index, build := range builds {
		matrix.Builds = append(matrix.Builds, StageBuild{
			Number:      build.Number,
			Result:      build.Result,
			Status:      string(build.Status),
			Timestamp:   build.Timestamp,
			Duration:    build.Duration,
			CommitCount: build.CommitCount,
			Stages:      stages[index],
		})
		for _, stage := range stages[index] {
			position, ok := positions[stage.Name]
			if !ok {
				continue
			}
			if stage.Status == "success" && stage.DurationMillis > 0 {
				sums[position] += stage.DurationMillis
				counts[position]++
			}
		}
	}

	matrix.Averages = make([]StageAverage, 0, len(matrix.StageNames))
	for index, name := range matrix.StageNames {
		average := int64(0)
		if counts[index] > 0 {
			average = sums[index] / int64(counts[index])
		}
		matrix.Averages = append(matrix.Averages, StageAverage{Name: name, Duration: average})
		matrix.AverageRun += average
	}
	return matrix, nil
}
