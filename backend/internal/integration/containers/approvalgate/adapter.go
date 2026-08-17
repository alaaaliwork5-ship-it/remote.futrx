// Package approvalgate provisions the human-approval gate into project
// containers: a Claude Code PreToolUse hook that pauses destructive shell
// commands until the Remote host's owner approves or denies them in the chat.
package approvalgate

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/integration/containers/assets"
	"github.com/futrx-com/remote.futrx.com/internal/integration/containers/command"
)

const (
	containerScriptsDir = "/workspace/scripts"

	// ContainerGateScript is the hook executable claude runs before Bash tools.
	ContainerGateScript = containerScriptsDir + "/remote-gate"
	// ContainerClaudeSettings is the claude settings file passed via --settings
	// that wires the PreToolUse hook.
	ContainerClaudeSettings = containerScriptsDir + "/remote-claude-settings.json"

	containerGateScriptHash      = containerScriptsDir + "/.remote-gate.sha256"
	containerClaudeSettingsHash  = containerScriptsDir + "/.remote-claude-settings.sha256"

	ensureTimeout = 10 * time.Second
)

//go:embed assets/remote-gate
var gateScript []byte

//go:embed assets/claude-settings.json
var claudeSettings []byte

// Adapter publishes the gate hook and its claude settings into project
// containers under /workspace so they survive replacement of the container.
type Adapter struct {
	runner    command.Runner
	publisher *assets.Publisher
}

func NewAdapter(runner command.Runner, publisher *assets.Publisher) *Adapter {
	return &Adapter{
		runner:    runner,
		publisher: publisher,
	}
}

// Ensure idempotently provisions the gate hook and claude settings.
func (a *Adapter) Ensure(ctx context.Context, containerName string) error {
	if !a.runner.Available() {
		return command.ErrUnavailable
	}

	out, err := command.RunWithTimeout(
		ctx,
		a.runner,
		ensureTimeout,
		"exec",
		containerName,
		"--",
		"install",
		"-d",
		"-m",
		"755",
		containerScriptsDir,
	)
	if err != nil {
		return fmt.Errorf("create approval-gate directory: %w; output: %s", err, out)
	}

	if err := a.publisher.PushVerified(
		ctx,
		containerName,
		gateScript,
		containerGateScriptHash,
		"755",
		ContainerGateScript,
	); err != nil {
		return fmt.Errorf("publish remote-gate hook: %w", err)
	}

	if err := a.publisher.PushVerified(
		ctx,
		containerName,
		claudeSettings,
		containerClaudeSettingsHash,
		"644",
		ContainerClaudeSettings,
	); err != nil {
		return fmt.Errorf("publish remote-gate claude settings: %w", err)
	}
	return nil
}
