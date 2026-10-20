package jenkins

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// TriggerBuild 触发构建；有参数时走 buildWithParameters，返回队列项 ID。
func (c *Client) TriggerBuild(ctx context.Context, fullName string, params url.Values) (int, error) {
	jobPath, err := JobPath(fullName)
	if err != nil {
		return 0, &APIError{Kind: ErrInvalidConfig, Message: err.Error()}
	}

	path := jobPath + "/build"
	if len(params) > 0 {
		path += "WithParameters"
	} else {
		params = url.Values{}
	}

	response, err := c.PostForm(ctx, path, params)
	if err != nil {
		return 0, err
	}
	return parseQueueID(response.Header.Get("Location")), nil
}

// StopBuild 停止指定构建。
func (c *Client) StopBuild(ctx context.Context, fullName string, number int) error {
	jobPath, err := JobPath(fullName)
	if err != nil {
		return &APIError{Kind: ErrInvalidConfig, Message: err.Error()}
	}
	if number <= 0 {
		return &APIError{Kind: ErrInvalidConfig, Message: "build number must be positive"}
	}

	_, err = c.PostForm(ctx, fmt.Sprintf("%s/%d/stop", jobPath, number), url.Values{})
	return err
}

// parseQueueID 从 Location（如 /queue/item/42/）里解析队列项 ID。
func parseQueueID(location string) int {
	const marker = "/queue/item/"
	index := strings.Index(location, marker)
	if index < 0 {
		return 0
	}
	rest := location[index+len(marker):]
	if end := strings.IndexAny(rest, "/?#"); end >= 0 {
		rest = rest[:end]
	}
	id, err := strconv.Atoi(rest)
	if err != nil {
		return 0
	}
	return id
}
