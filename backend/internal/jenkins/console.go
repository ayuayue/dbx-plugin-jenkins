package jenkins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const maxConsoleChunk = 512 * 1024

// GetConsole 读取从 start 开始的增量控制台日志。
// Jenkins 用 X-Text-Size 表示日志总长度、X-More-Data 表示是否还有后续内容。
func (c *Client) GetConsole(ctx context.Context, fullName string, number int, start int64) (*ConsoleChunk, error) {
	jobPath, err := JobPath(fullName)
	if err != nil {
		return nil, &APIError{Kind: ErrInvalidConfig, Message: err.Error()}
	}
	if number <= 0 {
		return nil, &APIError{Kind: ErrInvalidConfig, Message: "build number must be positive"}
	}
	if start < 0 {
		start = 0
	}

	path := fmt.Sprintf("%s/%d/logText/progressiveText?start=%d", jobPath, number, start)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.resolve(path), nil)
	if err != nil {
		return nil, &APIError{Kind: ErrRequest, Message: err.Error()}
	}
	request.Header.Set("Accept", "text/plain")

	response, err := c.do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxConsoleChunk+1))
	if readErr != nil {
		return nil, &APIError{Kind: ErrNetwork, Message: "cannot read Jenkins console output"}
	}
	truncated := len(body) > maxConsoleChunk
	if truncated {
		body = body[:maxConsoleChunk]
	}
	if response.StatusCode >= 400 {
		return nil, newAPIError(response.StatusCode, body)
	}

	nextStart := start + int64(len(body))
	headerSize, _ := strconv.ParseInt(strings.TrimSpace(response.Header.Get("X-Text-Size")), 10, 64)
	if !truncated && headerSize > 0 {
		nextStart = headerSize
	}

	moreData := truncated ||
		strings.EqualFold(response.Header.Get("X-More-Data"), "true") ||
		headerSize > nextStart

	return &ConsoleChunk{
		Text:      string(body),
		NextStart: nextStart,
		MoreData:  moreData,
	}, nil
}
