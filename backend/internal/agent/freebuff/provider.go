package freebuff

import (
	"context"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
	"github.com/futrx-com/remote.futrx.com/internal/agent/provisioning"
	serviceproject "github.com/futrx-com/remote.futrx.com/internal/service/project"
)

// Provider adapts the Freebuff CLI as a provider entry. Because the CLI is
// interactive-only, Run does not execute a process: it emits the terminal
// guidance as an assistant message and completes, so a prompt sent with the
// Freebuff provider reads as instructions rather than a hard error.
type Provider struct{}

// New matches the catalog's provider constructor shape; Freebuff never runs
// inside a project container process itself, so the arguments are unused.
func New(_ *serviceproject.Service, _ provisioning.ContainerDependencies) *Provider {
	return &Provider{}
}

func (p *Provider) ID() agent.ProviderID {
	return agent.ProviderFreebuff
}

func (p *Provider) Parser(agent.RunRequest) agent.LineParser {
	return noopParser{}
}

func (p *Provider) Run(_ context.Context, req agent.RunRequest, emit func(agent.Event)) error {
	if emit == nil {
		emit = func(agent.Event) {}
	}
	emit(agent.Event{
		T:              time.Now().UnixMilli(),
		Type:           agent.EventAssistantTextDelta,
		Provider:       agent.ProviderFreebuff,
		ConversationID: req.ConversationID,
		ItemKind:       agent.ItemMessage,
		Text:           terminalHint,
	})
	emit(agent.Event{
		T:              time.Now().UnixMilli(),
		Type:           agent.EventRunCompleted,
		Provider:       agent.ProviderFreebuff,
		ConversationID: req.ConversationID,
	})
	return nil
}

// noopParser satisfies the Provider interface; no process output is ever
// streamed for Freebuff.
type noopParser struct{}

func (noopParser) ParseLine([]byte) ([]agent.Event, error) {
	return nil, nil
}
