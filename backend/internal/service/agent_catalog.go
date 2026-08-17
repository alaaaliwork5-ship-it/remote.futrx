package service

import (
	"github.com/futrx-com/remote.futrx.com/internal/agent"
	antigravityagent "github.com/futrx-com/remote.futrx.com/internal/agent/antigravity"
	claudeagent "github.com/futrx-com/remote.futrx.com/internal/agent/claude"
	codexagent "github.com/futrx-com/remote.futrx.com/internal/agent/codex"
	freebuffagent "github.com/futrx-com/remote.futrx.com/internal/agent/freebuff"
	kimiagent "github.com/futrx-com/remote.futrx.com/internal/agent/kimi"
	opencodeagent "github.com/futrx-com/remote.futrx.com/internal/agent/opencode"
	"github.com/futrx-com/remote.futrx.com/internal/agent/provisioning"
	agentauth "github.com/futrx-com/remote.futrx.com/internal/service/agent/auth"
	serviceproject "github.com/futrx-com/remote.futrx.com/internal/service/project"
)

// agentDefinition is the single composition catalog for one agent. Adding an
// agent means adding one entry with its provider, provisioning profile, and
// shared-auth caller configuration. Name and description are display metadata
// served to the frontend via the agent catalog endpoint.
type agentDefinition struct {
	name        string
	description string
	profile     func() provisioning.Profile
	provider    func(*serviceproject.Service, provisioning.ContainerDependencies) agent.Provider
	authBinding func() agentauth.Binding
}

func agentDefinitions() []agentDefinition {
	return []agentDefinition{
		{
			name:        "Claude",
			description: "Anthropic's Claude Code. Starts `claude auth login --claudeai` on the host and signs in once with your Anthropic subscription; tokens seed into every container.",
			profile:     claudeagent.Profile,
			provider: func(projects *serviceproject.Service, containerDeps provisioning.ContainerDependencies) agent.Provider {
				return claudeagent.New(projects, containerDeps)
			},
			authBinding: func() agentauth.Binding {
				return agentauth.NewCodeBinding(agent.ProviderClaude, claudeagent.NewAuth())
			},
		},
		{
			name:        "Codex",
			description: "OpenAI's Codex CLI. Starts `codex login --device-auth` on the host and signs in with ChatGPT so prompts use subscription limits instead of API-key billing.",
			profile:     codexagent.Profile,
			provider: func(projects *serviceproject.Service, containerDeps provisioning.ContainerDependencies) agent.Provider {
				return codexagent.New(projects, containerDeps)
			},
			authBinding: func() agentauth.Binding {
				return agentauth.NewDeviceBinding(agent.ProviderCodex, codexagent.NewAuth())
			},
		},
		{
			name:        "Kimi",
			description: "Moonshot's Kimi Code. Starts `kimi login` on the host and signs in with your Kimi subscription via a device code — no API key, billed against your membership quota.",
			profile:     kimiagent.Profile,
			provider: func(projects *serviceproject.Service, containerDeps provisioning.ContainerDependencies) agent.Provider {
				return kimiagent.New(projects, containerDeps)
			},
			authBinding: func() agentauth.Binding {
				return agentauth.NewDeviceBinding(agent.ProviderKimi, kimiagent.NewAuth())
			},
		},
		{
			name:        "Antigravity",
			description: "Google's Antigravity CLI. No host-side login flow — sign in once per workspace by running `agy` in a chat Terminal.",
			profile:     antigravityagent.Profile,
			provider: func(projects *serviceproject.Service, containerDeps provisioning.ContainerDependencies) agent.Provider {
				return antigravityagent.New(projects, containerDeps)
			},
			// agy's bare-launch sign-in never exits its TUI, so it cannot run
			// under the shared code/device auth services. The binding is
			// registered without a service (reports unavailable); sign-in is a
			// one-time `agy` run in the chat terminal per workspace.
			authBinding: func() agentauth.Binding {
				return agentauth.NewCodeBinding(agent.ProviderAntigravity, nil)
			},
		},
		{
			name:        "OpenCode",
			description: "Open-source coding agent. Add an Anthropic, OpenAI, or Google API key — it is stored in OpenCode's auth.json and shared with project containers.",
			profile:     opencodeagent.Profile,
			provider: func(projects *serviceproject.Service, containerDeps provisioning.ContainerDependencies) agent.Provider {
				return opencodeagent.New(projects, containerDeps)
			},
		authBinding: func() agentauth.Binding {
			return agentauth.NewAPIKeyBinding(agent.ProviderOpenCode, opencodeagent.NewAuth())
		},
		},
		{
			name:        "Freebuff",
			description: "Free, ad-supported coding agent — no sign-in required. Run `freebuff` in a chat Terminal to use it interactively.",
			profile:     freebuffagent.Profile,
			provider: func(projects *serviceproject.Service, containerDeps provisioning.ContainerDependencies) agent.Provider {
				return freebuffagent.New(projects, containerDeps)
			},
			// freebuff needs no host credentials (free, ad-supported) and its
			// CLI is interactive-only, so there is no host auth flow to run.
			// The binding is registered without a service; sign-in (optional)
			// happens inside the freebuff TUI in the chat terminal.
			authBinding: func() agentauth.Binding {
				return agentauth.NewCodeBinding(agent.ProviderFreebuff, nil)
			},
		},
	}
}

// AgentInfo is the transport-facing description of one catalog agent: display
// metadata plus the live auth state derived from its shared-auth binding.
type AgentInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	AuthMethod    string `json:"authMethod"`
	AuthAvailable bool   `json:"authAvailable"`
	Authenticated bool   `json:"authenticated"`
}

// describeAgent merges a catalog definition's display metadata with its auth
// binding's runtime state. Agents without a usable host-side flow (available
// binding) report "terminal" so the frontend can render a terminal sign-in
// card instead of a host login form.
func describeAgent(def agentDefinition, binding agentauth.Binding) AgentInfo {
	authMethod := "terminal"
	if binding.Available() {
		authMethod = string(binding.Flow())
	}
	return AgentInfo{
		ID:            string(binding.ID()),
		Name:          def.name,
		Description:   def.description,
		AuthMethod:    authMethod,
		AuthAvailable: binding.Available(),
		Authenticated: binding.Authenticated(),
	}
}

// DescribeAgentCatalog renders the full agent list in catalog order, pairing
// each definition with the binding registered for the same provider.
func DescribeAgentCatalog(definitions []agentDefinition, bindings []agentauth.Binding) []AgentInfo {
	catalog := make([]AgentInfo, 0, len(definitions))
	for index, definition := range definitions {
		if index >= len(bindings) {
			break
		}
		catalog = append(catalog, describeAgent(definition, bindings[index]))
	}
	return catalog
}

// AgentProfiles returns the container-facing profiles from the same catalog
// used to register providers and their auth callers.
func AgentProfiles() []provisioning.Profile {
	return profilesFromDefinitions(agentDefinitions())
}

func profilesFromDefinitions(definitions []agentDefinition) []provisioning.Profile {
	profiles := make([]provisioning.Profile, 0, len(definitions))
	for _, definition := range definitions {
		profiles = append(profiles, definition.profile())
	}
	return profiles
}
