package narrate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

// runDir is a one-year run with one generator failure.
func runDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range map[string]string{
		"run.json":     `{"seed":3,"scenario":{"name":"tiny","warmupYears":0,"years":1,"modules":["demography"]}}`,
		"events.jsonl": `{"day":100,"type":"failure","outageDays":2}` + "\n",
		"daily.csv":    "day,population\n1,10\n365,11\n",
		"yearly.csv":   "day,medianAge\n365,30\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// fake answers one Messages request with a streamed reply of text and
// stop reason, recording the request it got.
func fake(t *testing.T, text, stop string, got *map[string]any, header *http.Header) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, got); err != nil {
			t.Errorf("request body: %v", err)
		}
		*header = r.Header.Clone()
		w.Header().Set("Content-Type", "text/event-stream")
		send := func(kind string, data any) {
			b, _ := json.Marshal(data)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, b)
		}
		send("message_start", map[string]any{"type": "message_start", "message": map[string]any{"id": "msg_1", "type": "message", "role": "assistant",
			"model": "claude-opus-5-5", "content": []any{}, "stop_reason": nil, "usage": map[string]any{"input_tokens": 10, "output_tokens": 0}}})
		send("content_block_start", map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
		send("content_block_delta", map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": text}})
		send("content_block_stop", map[string]any{"type": "content_block_stop", "index": 0})
		send("message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": stop}, "usage": map[string]any{"output_tokens": 5}})
		send("message_stop", map[string]any{"type": "message_stop"})
	}))
}

func opts(url string) []option.RequestOption {
	return []option.RequestOption{option.WithBaseURL(url), option.WithAPIKey("test"), option.WithMaxRetries(0)}
}

var day = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

// The request carries the model, the system prompt, the chronicle as the
// only message and the default fallback; the reply lands in story.md
// under its header.
func TestWrite(t *testing.T) {
	dir := runDir(t)
	var got map[string]any
	var header http.Header
	srv := fake(t, "# The first year\n\nThe generator failed.", "end_turn", &got, &header)
	defer srv.Close()
	if err := Write(context.Background(), dir, Model, day, opts(srv.URL)...); err != nil {
		t.Fatal(err)
	}
	if got["model"] != Model || got["fallbacks"] != "default" || !strings.Contains(header.Get("anthropic-beta"), "server-side-fallback-2026-07-01") {
		t.Errorf("request: model %v, fallbacks %v, beta %q", got["model"], got["fallbacks"], header.Get("anthropic-beta"))
	}
	if sys, _ := json.Marshal(got["system"]); !strings.Contains(string(sys), "archivist of Silo 18") {
		t.Errorf("system prompt not sent: %s", sys)
	}
	msgs, _ := json.Marshal(got["messages"])
	if !strings.Contains(string(msgs), "The generator failed: 2 days without power.") || strings.Count(string(msgs), `"role"`) != 1 {
		t.Errorf("messages are not the chronicle alone: %s", msgs)
	}
	data, err := os.ReadFile(filepath.Join(dir, "story.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "*Written by claude-opus-5-5 on 2026-11-04 from this run's chronicle. The model may err; the chronicle below is the record.*\n\n# The first year\n\nThe generator failed.\n"
	if string(data) != want {
		t.Errorf("story.md:\n%q\nwant\n%q", data, want)
	}
}

// A refusal, a cut-off reply or an API error writes no story.md.
func TestNoStory(t *testing.T) {
	for _, stop := range []string{"refusal", "max_tokens"} {
		dir := runDir(t)
		var got map[string]any
		var header http.Header
		srv := fake(t, "partial", stop, &got, &header)
		if err := Write(context.Background(), dir, Model, day, opts(srv.URL)...); err == nil {
			t.Errorf("%s: no error", stop)
		}
		srv.Close()
		if _, err := os.Stat(filepath.Join(dir, "story.md")); err == nil {
			t.Errorf("%s: story.md written", stop)
		}
	}
	dir := runDir(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	}))
	defer srv.Close()
	err := Write(context.Background(), dir, Model, day, opts(srv.URL)...)
	if err == nil || !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("401: %v, want the credentials hint", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "story.md")); err == nil {
		t.Error("401: story.md written")
	}
}

// Haiku has no server-side fallback, so none is asked for.
func TestHaikuNoFallback(t *testing.T) {
	var got map[string]any
	var header http.Header
	srv := fake(t, "ok", "end_turn", &got, &header)
	defer srv.Close()
	if err := Write(context.Background(), runDir(t), "claude-haiku-5-5", day, opts(srv.URL)...); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["fallbacks"]; ok {
		t.Errorf("fallbacks sent for Haiku: %v", got["fallbacks"])
	}
}

// With no credentials anywhere, Write fails before sending a request and
// writes no story.md.
func TestNoCredentials(t *testing.T) {
	for _, k := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_PROFILE", "ANTHROPIC_FEDERATION_RULE_ID", "ANTHROPIC_IDENTITY_TOKEN"} {
		t.Setenv(k, "") // restored after the test
		os.Unsetenv(k)
	}
	home := t.TempDir() // no profile file
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer srv.Close()
	dir := runDir(t)
	if err := Write(context.Background(), dir, Model, day, option.WithBaseURL(srv.URL), option.WithMaxRetries(0)); err == nil {
		t.Error("no error without credentials")
	}
	if requests != 0 {
		t.Errorf("%d requests sent without credentials", requests)
	}
	if _, err := os.Stat(filepath.Join(dir, "story.md")); err == nil {
		t.Error("story.md written without credentials")
	}
}
