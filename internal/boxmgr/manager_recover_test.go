package boxmgr

import (
	"strings"
	"testing"
)

// recoverWrap mirrors the defer/recover contract used by newBoxRecover.
// It guarantees that a panic in the wrapped call becomes an error instead of
// crashing the process, which is what protects startup from malformed nodes
// that make the sing-box library panic during outbound initialization.
func recoverWrap(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &recoveredError{value: r}
		}
	}()
	return fn()
}

type recoveredError struct{ value any }

func (e *recoveredError) Error() string { return "recovered panic" }

func TestRecoverWrap_ConvertsPanicToError(t *testing.T) {
	err := recoverWrap(func() error {
		panic("unknown value")
	})
	if err == nil {
		t.Fatalf("expected error from panic, got nil")
	}
	if !strings.Contains(err.Error(), "recovered panic") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecoverWrap_PassesThroughNormalError(t *testing.T) {
	sentinel := &recoveredError{value: "boom"}
	err := recoverWrap(func() error {
		return sentinel
	})
	if err != sentinel {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestRecoverWrap_NoErrorOnSuccess(t *testing.T) {
	if err := recoverWrap(func() error { return nil }); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
