package jenkins

import "strings"

// Status 是 Jenkins color 归一化后的构建/任务状态。
type Status string

const (
	StatusSuccess  Status = "success"
	StatusFailure  Status = "failure"
	StatusUnstable Status = "unstable"
	StatusBuilding Status = "building"
	StatusAborted  Status = "aborted"
	StatusDisabled Status = "disabled"
	StatusNotBuilt Status = "not_built"
	StatusUnknown  Status = "unknown"
)

// NormalizeColor 把 Jenkins 的 color 字段（blue、red_anime 等）归一化为 Status。
// blue/red/yellow/grey/aborted 带 _anime 后缀表示正在构建；未知颜色一律 unknown。
func NormalizeColor(color string) Status {
	normalized := strings.ToLower(strings.TrimSpace(color))
	if normalized == "" {
		return StatusUnknown
	}

	building := strings.HasSuffix(normalized, "_anime")
	base := strings.TrimSuffix(normalized, "_anime")

	var status Status
	switch base {
	case "blue":
		status = StatusSuccess
	case "red":
		status = StatusFailure
	case "yellow":
		status = StatusUnstable
	case "grey", "notbuilt":
		status = StatusNotBuilt
	case "disabled":
		status = StatusDisabled
	case "aborted":
		status = StatusAborted
	default:
		return StatusUnknown
	}

	if status == StatusDisabled {
		return StatusDisabled
	}
	if building {
		return StatusBuilding
	}
	return status
}
