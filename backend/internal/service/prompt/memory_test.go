package prompt

import (
	"strings"
	"testing"
)

func TestPromptWithProjectMemory(t *testing.T) {
	base := "Do the thing."
	withMemory := promptWithProjectMemory("Use Go 1.25.", base)
	if !strings.Contains(withMemory, "Project memory") {
		t.Fatalf("missing header: %q", withMemory)
	}
	if !strings.Contains(withMemory, "Use Go 1.25.") {
		t.Fatalf("missing memory content: %q", withMemory)
	}
	if !strings.HasSuffix(withMemory, "Current request:\n"+base) {
		t.Fatalf("prompt must end with the current request: %q", withMemory)
	}
}

func TestPromptWithProjectMemoryEmpty(t *testing.T) {
	if got := promptWithProjectMemory("   ", "Do it."); got != "Do it." {
		t.Fatalf("empty memory must not alter prompt, got %q", got)
	}
}

func TestPromptWithProjectMemoryTruncates(t *testing.T) {
	long := strings.Repeat("x", 30000)
	withMemory := promptWithProjectMemory(long, "Do it.")
	if !strings.Contains(withMemory, "[...memory truncated]") {
		t.Fatalf("expected truncation marker")
	}
	// The original prompt must survive intact at the end.
	if !strings.HasSuffix(withMemory, "Current request:\nDo it.") {
		t.Fatalf("prompt lost the request: %q", withMemory[:120])
	}
}
