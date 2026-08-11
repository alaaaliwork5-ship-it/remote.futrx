// Package minimax runs MiniMax models through the Codex CLI.
//
// MiniMax publishes a Codex-compatible endpoint, so this provider reuses the
// Codex binary and its `exec --json` event stream rather than shipping a second
// CLI. Everything else is its own: a static API key instead of a ChatGPT
// subscription grant, and a separate CODEX_HOME so configuration, session
// rollouts, and history never mix with the Codex agent's.
package minimax

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
	codexagent "github.com/futrx-com/remote.futrx.com/internal/agent/codex"
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
	auth          *Auth
}

func New(projects ProjectResolver, containerDeps provisioning.ContainerDependencies) *Provider {
	return &Provider{
		projects:      projects,
		containerDeps: containerDeps,
		profile:       Profile(),
		// The stored key file is the single source of truth, so this instance
		// only ever reads it. The auth service registered by the catalog owns
		// mutations and subscriber notification.
		auth: NewAuth(),
	}
}

func (p *Provider) ID() agent.ProviderID {
	return agent.ProviderMiniMax
}

// Parser reuses the Codex stream parser: MiniMax speaks the Responses wire API
// through the same binary, so `exec --json` emits the same event shapes. Events
// are tagged with this provider's ID via the request.
func (p *Provider) Parser(req agent.RunRequest) agent.LineParser {
	if req.Provider == "" {
		req.Provider = agent.ProviderMiniMax
	}
	return codexagent.NewParser(req)
}

func (p *Provider) Run(ctx context.Context, req agent.RunRequest, emit func(agent.Event)) error {
	if emit == nil {
		emit = func(agent.Event) {}
	}
	if req.Provider == "" {
		req.Provider = agent.ProviderMiniMax
	}

	if req.Fork && req.ResumeID != "" {
		if newID, ferr := p.forkSession(ctx, req); ferr != nil {
			log.Printf("minimax[%s] fork session: %v — starting fresh", req.ConversationID, ferr)
			req.ResumeID = ""
		} else {
			req.ResumeID = newID
			emit(agent.Event{
				T:              time.Now().UnixMilli(),
				Type:           agent.EventSessionUpdated,
				Provider:       agent.ProviderMiniMax,
				ConversationID: req.ConversationID,
				SessionID:      newID,
			})
		}
	}
	req.Fork = false

	cmd, _, err := p.buildCmd(ctx, req, p.args(req), emit)
	if err != nil {
		return err
	}
	err = agentruntime.RunProcess(ctx, cmd, p.Parser(req), emit, agentruntime.ProcessOptions{
		Name:           "minimax",
		LogID:          req.ConversationID,
		Provider:       agent.ProviderMiniMax,
		ConversationID: req.ConversationID,
	})
	if err != nil && req.ResumeID != "" && strings.Contains(strings.ToLower(agentruntime.ErrorStderr(err)), "no rollout found") {
		return fmt.Errorf("%w: %s", agent.ErrSessionNotFound, strings.TrimSpace(agentruntime.ErrorStderr(err)))
	}
	// No credential sync-back: the API key lives only on the host, so a run can
	// never mutate it the way a refreshed OAuth token would.
	return err
}

// forkSession duplicates the parent rollout under a fresh session id so a
// forked chat continues from the same history without mutating the parent's
// transcript. Codex's headless `exec` has no fork primitive, so we copy the
// rollout file: the parent's uuid (unique) is rewritten everywhere in the file
// and in its name, yielding a session codex can resume by id.
//
// The search is rooted at MiniMax's own sessions directory, so a fork can never
// reach into — or collide with — the Codex agent's rollouts.
func (p *Provider) forkSession(ctx context.Context, req agent.RunRequest) (string, error) {
	newID, err := newUUID()
	if err != nil {
		return "", err
	}
	parent := req.ResumeID

	var cmd *exec.Cmd
	if req.ProjectID != "" && p.projects != nil {
		project, perr := p.projects.Get(ctx, serviceproject.ID(req.ProjectID))
		if perr != nil {
			return "", perr
		}
		if project.ContainerName == "" {
			return "", fmt.Errorf("project %s has no container", project.ID)
		}
		script := forkScript(containerMiniMaxHome+"/sessions", parent, newID)
		cmd = exec.CommandContext(ctx, "lxc", "exec", project.ContainerName, "--", "sh", "-c", script)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", forkScript(hostSessionsDir(), parent, newID))
	}
	if out, cerr := cmd.CombinedOutput(); cerr != nil {
		return "", fmt.Errorf("copy rollout: %w: %s", cerr, strings.TrimSpace(string(out)))
	}
	return newID, nil
}

func forkScript(sessionsDir, parent, newID string) string {
	return fmt.Sprintf(`set -e
src=$(find %[1]s -name '*-%[2]s.jsonl' 2>/dev/null | head -1)
[ -n "$src" ] || { echo NOTFOUND >&2; exit 3; }
dest="$(dirname "$src")/$(basename "$src" | sed 's/%[2]s/%[3]s/')"
sed 's/%[2]s/%[3]s/g' "$src" > "$dest"`, sessionsDir, parent, newID)
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
