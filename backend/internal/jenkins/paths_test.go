package jenkins

import "testing"

func TestBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		scheme  string
		host    string
		port    int
		base    string
		want    string
		wantErr bool
	}{
		{
			name:   "https with explicit port",
			scheme: "https",
			host:   "ci.example.com",
			port:   443,
			want:   "https://ci.example.com:443",
		},
		{
			name:   "http with base path and trailing slash",
			scheme: "http",
			host:   "10.0.0.5",
			port:   8080,
			base:   "/jenkins/",
			want:   "http://10.0.0.5:8080/jenkins",
		},
		{
			name:   "base path without leading slash",
			scheme: "https",
			host:   "ci.example.com",
			port:   8443,
			base:   "jenkins",
			want:   "https://ci.example.com:8443/jenkins",
		},
		{
			name:   "nested base path",
			scheme: "https",
			host:   "ci.example.com",
			port:   443,
			base:   "/a/b/",
			want:   "https://ci.example.com:443/a/b",
		},
		{
			name:   "uppercase scheme is normalized",
			scheme: "HTTPS",
			host:   "CI.EXAMPLE.COM",
			port:   443,
			want:   "https://CI.EXAMPLE.COM:443",
		},
		{
			name:   "ipv6 host is bracketed",
			scheme: "http",
			host:   "::1",
			port:   8080,
			want:   "http://[::1]:8080",
		},
		{
			name:    "empty host",
			scheme:  "https",
			host:    "   ",
			port:    443,
			wantErr: true,
		},
		{
			name:    "unsupported scheme",
			scheme:  "ftp",
			host:    "ci.example.com",
			port:    21,
			wantErr: true,
		},
		{
			name:    "host already contains scheme",
			scheme:  "https",
			host:    "https://ci.example.com",
			port:    443,
			wantErr: true,
		},
		{
			name:    "host contains path",
			scheme:  "https",
			host:    "ci.example.com/jenkins",
			port:    443,
			wantErr: true,
		},
		{
			name:    "port zero",
			scheme:  "https",
			host:    "ci.example.com",
			port:    0,
			wantErr: true,
		},
		{
			name:    "port out of range",
			scheme:  "https",
			host:    "ci.example.com",
			port:    70000,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := BaseURL(test.scheme, test.host, test.port, test.base)
			if test.wantErr {
				if err == nil {
					t.Fatalf("BaseURL() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("BaseURL() error = %v", err)
			}
			if got != test.want {
				t.Errorf("BaseURL() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestJobPath(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		want     string
		wantErr  bool
	}{
		{name: "simple job", fullName: "build", want: "/job/build"},
		{name: "nested folder", fullName: "team/api/build", want: "/job/team/job/api/job/build"},
		{name: "surrounding slashes", fullName: "/team/build/", want: "/job/team/job/build"},
		{name: "segments are trimmed", fullName: " team / api ", want: "/job/team/job/api"},
		{
			name:     "spaces and hash are escaped",
			fullName: "release 1.0/构建 #1",
			want:     "/job/release%201.0/job/%E6%9E%84%E5%BB%BA%20%231",
		},
		{name: "empty", fullName: "", wantErr: true},
		{name: "only slash", fullName: "/", wantErr: true},
		{name: "empty segment", fullName: "a//b", wantErr: true},
		{name: "parent segment", fullName: "a/../b", wantErr: true},
		{name: "current segment", fullName: ".", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := JobPath(test.fullName)
			if test.wantErr {
				if err == nil {
					t.Fatalf("JobPath() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("JobPath() error = %v", err)
			}
			if got != test.want {
				t.Errorf("JobPath() = %q, want %q", got, test.want)
			}
		})
	}
}
