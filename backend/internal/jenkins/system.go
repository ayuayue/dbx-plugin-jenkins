package jenkins

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const queueTree = "items[id,task[name,url],why,stuck,blocked,buildableIn,inQueueSince,actions[causes[shortDescription]]]"

const nodesTree = "computer[displayName,offline,temporarilyOffline,numExecutors,idle,offlineCauseReason,assignedLabels[name],executors[idle]]"

// executorsTree 取每个槽位上正在跑的可执行对象；Queue$Executable 形态只返回 url。
// 任务全名要从 url 反解：fullDisplayName 是「父 » 子」的展示名，不是 fullName。
const executorsTree = "computer[displayName,offline,numExecutors,executors[idle,currentExecutable[fullDisplayName,number,url,timestamp,estimatedDuration,duration,building,result]]]"

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

// GetExecutors 读取所有节点上的执行器槽位（空闲槽位也返回，供界面显示占位）。
func (c *Client) GetExecutors(ctx context.Context) ([]Executor, error) {
	var payload struct {
		Computer []struct {
			DisplayName string `json:"displayName"`
			Offline     bool   `json:"offline"`
			Executors   []struct {
				Idle              bool `json:"idle"`
				CurrentExecutable *struct {
					FullDisplayName   string `json:"fullDisplayName"`
					Number            int    `json:"number"`
					URL               string `json:"url"`
					Timestamp         int64  `json:"timestamp"`
					EstimatedDuration int64  `json:"estimatedDuration"`
					Duration          int64  `json:"duration"`
					Building          bool   `json:"building"`
					Result            string `json:"result"`
				} `json:"currentExecutable"`
			} `json:"executors"`
		} `json:"computer"`
	}
	query := url.Values{"tree": {executorsTree}}
	if err := c.Get(ctx, "/computer/api/json?"+query.Encode(), &payload); err != nil {
		return nil, err
	}

	rows := make([]Executor, 0)
	for _, node := range payload.Computer {
		for i, slot := range node.Executors {
			row := Executor{
				Node:        node.DisplayName,
				NodeOffline: node.Offline,
				Index:       i + 1,
				Idle:        slot.Idle,
			}
			if exec := slot.CurrentExecutable; exec != nil {
				row.JobName = jobFullNameFromURL(exec.URL)
				// url 缺失（极老版本）时才退回展示名，只能救顶层任务。
				if row.JobName == "" {
					row.JobName = jobNameFromDisplayName(exec.FullDisplayName, exec.Number)
				}
				row.BuildNumber = exec.Number
				row.Building = exec.Building
				row.Timestamp = exec.Timestamp
				row.EstimatedDuration = exec.EstimatedDuration
				row.Duration = exec.Duration
				row.Result = exec.Result
				// Queue$Executable：执行器已占位，但构建号与显示名都还没有。
				row.Pending = exec.Number == 0 && strings.TrimSpace(exec.FullDisplayName) == ""
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// jobFullNameFromURL 从构建 URL 反解任务全名（folder/job）。
//
// 不能拿 fullDisplayName 当 fullName：Jenkins 的 AbstractItem.getFullDisplayName()
// 是「父 » 子」的展示名（分隔符 U+00BB，还可能是自定义 displayName），
// 而 REST 路径要的是 getName() 用 "/" 拼出来的 fullName。
// 构建 URL 形如 <context>/job/<folder>/job/<job>/<number>/，按 job 段两两取名字即可。
func jobFullNameFromURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	names := make([]string, 0, len(segments)/2)
	for i := 0; i < len(segments)-1; i++ {
		if segments[i] == "job" {
			names = append(names, segments[i+1])
			i++
		}
	}
	return strings.Join(names, "/")
}

// jobNameFromDisplayName 从「任务全名 #构建号」里剥掉尾部构建号。
// 只对顶层任务可靠（文件夹任务用的是 » 分隔的展示名），仅作 url 缺失时的兜底。
// Run.getFullDisplayName() 的格式是 parent 显示名 + " #" + number；
// 自定义 displayName 不遵循该格式时原样返回，避免误截。
func jobNameFromDisplayName(display string, number int) string {
	trimmed := strings.TrimSpace(display)
	if trimmed == "" || number <= 0 {
		return trimmed
	}
	suffix := " #" + strconv.Itoa(number)
	if strings.HasSuffix(trimmed, suffix) {
		return strings.TrimSpace(strings.TrimSuffix(trimmed, suffix))
	}
	return trimmed
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
