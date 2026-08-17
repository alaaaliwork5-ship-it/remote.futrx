package opencode

import (
	"reflect"
	"testing"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
)

func TestVariantArgMapsReasoningEffortToOpenCodeVariants(t *testing.T) {
	tests := []struct {
		effort agent.ReasoningEffort
		want   string
	}{
		{effort: "", want: ""},
		{effort: "auto", want: ""},
		{effort: "none", want: "none"},
		{effort: "minimal", want: "minimal"},
		{effort: "low", want: "low"},
		{effort: "medium", want: "medium"},
		{effort: "high", want: "high"},
		{effort: "xhigh", want: "xhigh"},
		{effort: "max", want: "max"},
		// opencode has no ultra variant; omit so the model picks its default.
		{effort: "ultra", want: ""},
		// Case and whitespace tolerance mirrors the other providers.
		{effort: "  HIGH ", want: "high"},
		{effort: "Unknown", want: ""},
	}
	for _, test := range tests {
		if got := variantArg(test.effort); got != test.want {
			t.Errorf("variantArg(%q) = %q, want %q", test.effort, got, test.want)
		}
	}
}

func TestArgsIncludesVariantAndModelFlags(t *testing.T) {
	provider := &Provider{}
	args := provider.args(agent.RunRequest{
		Model: "anthropic/claude-sonnet-4-5",
		Preferences: agent.RunPreferences{
			ReasoningEffort: agent.ReasoningEffort("max"),
		},
	})
	want := []string{"run", "--format", "json", "--auto", "--model", "anthropic/claude-sonnet-4-5", "--variant", "max", ""}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func TestArgsOmitsVariantWhenAuto(t *testing.T) {
	provider := &Provider{}
	args := provider.args(agent.RunRequest{
		Model: "openai/gpt-5.4",
	})
	want := []string{"run", "--format", "json", "--auto", "--model", "openai/gpt-5.4", ""}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %v, want %v (no --variant when effort is Auto)", args, want)
	}
}
