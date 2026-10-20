package jenkins

import (
	"context"
	"fmt"
	"net/url"
)

const queueTree = "items[id,task[name,url],why,stuck,blocked,buildableIn,inQueueSince,actions[causes[shortDescription]]]"

const nodesTree = "computer[displayName,offline,temporarilyOffline,numExecutors,idle,offlineCauseReason,assignedLabels[name],executors[idle]]"

// GetOverview 聚合 Jenkins 概览（用户、模式、队列、节点、执行器）。
func (c *Client) GetOverview(ctx context.Context) (*Overview, error) {
	who, err := c.TestConnection(ctx)
	if err != nil {
		return nil, err
	}

	var info struct {
		Mode            string `json:"mode"`
		NodeDescription string `json:"nodeDescription"`
		NumExecutors    int    `json:"numExecutors"`
		QuietingDown    bool   `json:"quietingDown"`
	}
	query := url.Values{"tree": {"mode,nodeDescription,numExecutors,quietingDown"}}
	if err := c.Get(ctx, "/api/json?"+query.Encode(), &info); err != nil {
		return nil, err
	}

	queue, err := c.GetQueue(ctx)
	if err != nil {
		return nil, err
	}
	nodes, err := c.GetNodes(ctx)
	if err != nil {
		return nil, err
	}

	overview := &Overview{
		Version:         c.Version(),
		User:            who.Name,
		Mode:            info.Mode,
		NodeDescription: info.NodeDescription,
		QuietingDown:    info.QuietingDown,
		QueueLength:     len(queue),
		NodesTotal:      len(nodes),
	}
	for _, node := range nodes {
		if !node.Offline {
			overview.NodesOnline++
		}
		overview.ExecutorsTotal += node.NumExecutors
		overview.ExecutorsBusy += node.Busy
	}
	return overview, nil
}

// GetQueue 读取构建队列。
func (c *Client) GetQueue(ctx context.Context) ([]QueueItem, error) {
	var payload struct {
		Items []struct {
			ID   int `json:"id"`
			Task struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"task"`
			Why          string `json:"why"`
			Stuck        bool   `json:"stuck"`
			Blocked      bool   `json:"blocked"`
			BuildableIn  int64  `json:"buildableIn"`
			InQueueSince int64  `json:"inQueueSince"`
			Actions      []struct {
				Causes []Cause `json:"causes"`
			} `json:"actions"`
		} `json:"items"`
	}
	query := url.Values{"tree": {queueTree}}
	if err := c.Get(ctx, "/queue/api/json?"+query.Encode(), &payload); err != nil {
		return nil, err
	}

	items := make([]QueueItem, 0, len(payload.Items))
	for _, raw := range payload.Items {
		item := QueueItem{
			ID:           raw.ID,
			JobName:      raw.Task.Name,
			JobURL:       raw.Task.URL,
			Why:          raw.Why,
			Stuck:        raw.Stuck,
			Blocked:      raw.Blocked,
			BuildableIn:  raw.BuildableIn,
			InQueueSince: raw.InQueueSince,
		}
		for _, action := range raw.Actions {
			if len(action.Causes) > 0 {
				item.Cause = action.Causes[0].ShortDescription
				break
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// CancelQueueItem 取消指定队列项。
func (c *Client) CancelQueueItem(ctx context.Context, id int) error {
	if id <= 0 {
		return &APIError{Kind: ErrInvalidConfig, Message: "queue item id must be positive"}
	}
	_, err := c.PostForm(ctx, fmt.Sprintf("/queue/cancelItem?id=%d", id), url.Values{})
	return err
}

// GetNodes 读取节点列表。
func (c *Client) GetNodes(ctx context.Context) ([]Node, error) {
	var payload struct {
		Computer []struct {
			DisplayName        string `json:"displayName"`
			Offline            bool   `json:"offline"`
			TemporarilyOffline bool   `json:"temporarilyOffline"`
			NumExecutors       int    `json:"numExecutors"`
			Idle               bool   `json:"idle"`
			OfflineCauseReason string `json:"offlineCauseReason"`
			AssignedLabels     []struct {
				Name string `json:"name"`
			} `json:"assignedLabels"`
			Executors []struct {
				Idle bool `json:"idle"`
			} `json:"executors"`
		} `json:"computer"`
	}
	query := url.Values{"tree": {nodesTree}}
	if err := c.Get(ctx, "/computer/api/json?"+query.Encode(), &payload); err != nil {
		return nil, err
	}

	nodes := make([]Node, 0, len(payload.Computer))
	for _, raw := range payload.Computer {
		idleCount := 0
		for _, executor := range raw.Executors {
			if executor.Idle {
				idleCount++
			}
		}
		// 老版本可能没有 executors 明细，退化为控制器级 idle 布尔值。
		if len(raw.Executors) == 0 && raw.Idle {
			idleCount = 1
		}

		busy := raw.NumExecutors - idleCount
		if busy < 0 {
			busy = 0
		}
		node := Node{
			Name:               raw.DisplayName,
			Offline:            raw.Offline,
			TemporarilyOffline: raw.TemporarilyOffline,
			NumExecutors:       raw.NumExecutors,
			Idle:               idleCount,
			Busy:               busy,
			OfflineReason:      raw.OfflineCauseReason,
		}
		for _, label := range raw.AssignedLabels {
			node.Labels = append(node.Labels, label.Name)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}
