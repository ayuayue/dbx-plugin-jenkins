package jenkins

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// BaseURL 组装 Jenkins 根地址：scheme://host:port/basePath。
// 端口始终显式保留，避免 http/https 默认端口歧义；basePath 的首尾斜杠会被裁掉。
func BaseURL(scheme, host string, port int, basePath string) (string, error) {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q", scheme)
	}

	host = strings.TrimSpace(host)
	if host == "" {
		return "", errors.New("host is required")
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}
	if strings.Contains(host, "://") || strings.ContainsAny(host, "/?#") || strings.ContainsAny(host, " \t\r\n") {
		return "", fmt.Errorf("invalid host %q", host)
	}
	if port <= 0 || port > 65535 {
		return "", fmt.Errorf("invalid port %d", port)
	}

	base := strings.Trim(strings.TrimSpace(basePath), "/")
	if base != "" {
		for _, segment := range strings.Split(base, "/") {
			if segment == "" || segment == "." || segment == ".." {
				return "", fmt.Errorf("invalid base path %q", basePath)
			}
			if strings.ContainsAny(segment, "?#") {
				return "", fmt.Errorf("invalid base path %q", basePath)
			}
		}
	}

	target := url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
	}
	if base != "" {
		target.Path = "/" + base
	}
	return target.String(), nil
}

// JobPath 把 Jenkins fullName（如 folder/job）转成 REST 路径片段（如 /job/folder/job/job）。
// 每个路径段独立转义，空段与 "." / ".." 直接拒绝，避免拼出越界 URL。
func JobPath(fullName string) (string, error) {
	trimmed := strings.Trim(strings.TrimSpace(fullName), "/")
	if trimmed == "" {
		return "", errors.New("job name is required")
	}

	rawSegments := strings.Split(trimmed, "/")
	parts := make([]string, 0, len(rawSegments)*2)
	for _, segment := range rawSegments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return "", fmt.Errorf("job name contains an empty path segment: %q", fullName)
		}
		if segment == "." || segment == ".." {
			return "", fmt.Errorf("invalid path segment %q", segment)
		}
		parts = append(parts, "job", url.PathEscape(segment))
	}
	return "/" + strings.Join(parts, "/"), nil
}
