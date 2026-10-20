package jenkins

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetConsoleProgressive(t *testing.T) {
	var starts []string
	var accepts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/job/team/job/demo/12/logText/progressiveText" {
			http.NotFound(w, r)
			return
		}
		starts = append(starts, r.URL.Query().Get("start"))
		accepts = append(accepts, r.Header.Get("Accept"))
		switch r.URL.Query().Get("start") {
		case "0":
			w.Header().Set("X-Text-Size", "6")
			w.Header().Set("X-More-Data", "true")
			fmt.Fprint(w, "hello\n")
		case "6":
			w.Header().Set("X-Text-Size", "12")
			w.Header().Set("X-More-Data", "false")
			fmt.Fprint(w, "world\n")
		default:
			http.Error(w, "bad start", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	first, err := client.GetConsole(context.Background(), "team/demo", 12, 0)
	if err != nil {
		t.Fatalf("GetConsole(first) error = %v", err)
	}
	if first.Text != "hello\n" || first.NextStart != 6 || !first.MoreData {
		t.Errorf("first chunk = %+v, want hello/6/more", first)
	}

	second, err := client.GetConsole(context.Background(), "team/demo", 12, first.NextStart)
	if err != nil {
		t.Fatalf("GetConsole(second) error = %v", err)
	}
	if second.Text != "world\n" || second.NextStart != 12 || second.MoreData {
		t.Errorf("second chunk = %+v, want world/12/done", second)
	}

	if len(starts) != 2 || starts[0] != "0" || starts[1] != "6" {
		t.Errorf("starts = %+v, want [0 6]", starts)
	}
	for _, accept := range accepts {
		if accept != "text/plain" {
			t.Errorf("Accept = %q, want text/plain", accept)
		}
	}
}

func TestGetConsoleMissingTextSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "abc")
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, true)
	chunk, err := client.GetConsole(context.Background(), "demo", 1, 10)
	if err != nil {
		t.Fatalf("GetConsole() error = %v", err)
	}
	if chunk.NextStart != 13 {
		t.Errorf("NextStart = %d, want 13 (start + bytes)", chunk.NextStart)
	}
	if chunk.MoreData {
		t.Errorf("MoreData = true, want false")
	}
}
