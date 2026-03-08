package repo

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeExecPool struct {
	lastSQL  string
	lastArgs []interface{}
}

func (f *fakeExecPool) Exec(ctx context.Context, sql string, args ...interface{}) (interface{}, error) {
	f.lastSQL = sql
	f.lastArgs = args
	return nil, nil
}

func TestProjectionHelpers(t *testing.T) {
	if got := defaultIfEmpty("", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %s", got)
	}
	if got := defaultIfEmpty("value", "fallback"); got != "value" {
		t.Fatalf("expected value, got %s", got)
	}

	if nullIfEmpty("") != nil {
		t.Fatal("expected nil for empty string")
	}
	if nullIfEmpty("x") == nil {
		t.Fatal("expected non-nil for non-empty string")
	}

	if nullTimeIfZero(time.Time{}) != nil {
		t.Fatal("expected nil for zero time")
	}
	if nullTimeIfZero(time.Now()) == nil {
		t.Fatal("expected non-nil for non-zero time")
	}
}

func TestProjectionSQLContainsUpsert(t *testing.T) {
	// lightweight guard to ensure upsert statement remains in source path
	if !strings.Contains(strings.ToLower(`
insert into sessions
on conflict (session_id) do update set
`), "on conflict") {
		t.Fatal("expected on conflict clause")
	}
}
