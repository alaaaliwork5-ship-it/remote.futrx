package opencode

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
	"github.com/futrx-com/remote.futrx.com/internal/agent/provisioning"
	agentruntime "github.com/futrx-com/remote.futrx.com/internal/agent/runtime"
	serviceproject "github.com/futrx-com/remote.futrx.com/internal/service/project"
)

type ProjectResolver interface {
	Get(ctx context.Context, id serviceproject.ID) (serviceproject.Meta, error)
	Start(ctx context.Context, id serviceproject.ID) (serviceproject.Meta, error)
	ListSecrets(ctx context.Context, id serviceproject.ID) ([]serviceproject.Secret, error)
}

type Provider struct {
	projects      ProjectResolver
	containerDeps provisioning.ContainerDependencies
	profile       provisioning.Profile
}

func New(projects ProjectResolver, containerDeps provisioning.ContainerDependencies) *Provider {
	return &Provider{projects: projects, containerDeps: containerDeps, profile: Profile()}
}

func (p *Provider) ID() agent.ProviderID {
	return agent.ProviderOpenCode
}

func (p *Provider) Parser(req agent.RunRequest) agent.LineParser {
	return NewParser(req)
}

func (p *Provider) Run(ctx context.Context, req agent.RunRequest, emit func(agent.Event)) error {
	if emit == nil {
		emit = func(agent.Event) {}
	}
	if req.Provider == "" {
		req.Provider = agent.ProviderOpenCode
	}

	cmd, containerName, err := p.buildCmd(ctx, req, p.args(req), emit)
	if err != nil {
		return err
	}
	err = agentruntime.RunProcess(ctx, cmd, p.Parser(req), emit, agentruntime.ProcessOptions{
		Name:           "opencode",
		LogID:          req.ConversationID,
		Provider:       agent.ProviderOpenCode,
		ConversationID: req.ConversationID,
	})
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil
	}
	if err == nil && containerName != "" && p.containerDeps.Credentials != nil {
		syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if syncErr := p.containerDeps.Credentials.SyncFromContainer(syncCtx, containerName, p.profile.Credentials); syncErr != nil {
			log.Printf("opencode[%s] sync auth from %s: %v", req.ConversationID, containerName, syncErr)
		}
	}
	if err != nil {
		stderr := strings.TrimSpace(agentruntime.ErrorStderr(err))
		message := fmt.Sprintf("opencode run failed: %v", err)
		if stderr != "" {
			message = fmt.Sprintf("%s; output: %s", message, stderr)
		}
		if isAuthError(stderr) {
			message = "opencode is not authenticated — sign in from Settings → Agents, or run `opencode auth login` in this chat's Terminal, then retry"
		}
		emit(agent.Event{
			T:              time.Now().UnixMilli(),
			Type:           agent.EventRunFailed,
			Provider:       agent.ProviderOpenCode,
			ConversationID: req.ConversationID,
			Message:        message,
		})
		return agent.ErrRunFailed
	}
	// opencode exits when the session goes idle with no completion event, so
	// success is signaled here once the process exits cleanly.
	emit(agent.Event{
		T:              time.Now().UnixMilli(),
		Type:           agent.EventRunCompleted,
		Provider:       agent.ProviderOpenCode,
		ConversationID: req.ConversationID,
	})
	return nil
}

func isAuthError(output string) bool {
	lowered := strings.ToLower(output)
	return strings.Contains(lowered, "not authenticated") ||
		strings.Contains(lowered, "no credentials") ||
		strings.Contains(lowered, "authentication required") ||
		strings.Contains(lowered, "login required")
}
