package minimax

import (
	"strings"
	"testing"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
	codexagent "github.com/futrx-com/remote.futrx.com/internal/agent/codex"
)

func TestProfileIsIsolatedFromCodex(t *testing.T) {
	profile := Profile()
	codexProfile := codexagent.Profile()

	if profile.ID != string(agent.ProviderMiniMax) {
		t.Fatalf("profile ID = %q, want %q", profile.ID, agent.ProviderMiniMax)
	}

	// Every container-facing path must sit under MiniMax's own home so the two
	// agents cannot read each other's sessions, config, or credentials.
	if !strings.HasPrefix(profile.Instructions.Path, containerMiniMaxHome) {
		t.Fatalf("instructions path %q escapes %q", profile.Instructions.Path, containerMiniMaxHome)
	}
	if !strings.HasPrefix(profile.WorkspaceSkills.HomeSkillsDir, containerMiniMaxHome) {
		t.Fatalf("skills dir %q escapes %q", profile.WorkspaceSkills.HomeSkillsDir, containerMiniMaxHome)
	}
	if containerMiniMaxHome == "/root/.codex" {
		t.Fatal("MiniMax must not share the codex home")
	}
	if profile.WorkspaceSkills.WorkspaceHome == codexProfile.WorkspaceSkills.WorkspaceHome {
		t.Fatalf("workspace home %q collides with codex", profile.WorkspaceSkills.WorkspaceHome)
	}

	// Instructions deliberately share codex's hash path: the publisher batches
	// targets by hash, so this writes MiniMax's AGENTS.md in the same push.
	if profile.Instructions.HashPath != codexProfile.Instructions.HashPath {
		t.Fatalf("hash path = %q, want the shared %q",
			profile.Instructions.HashPath, codexProfile.Instructions.HashPath)
	}
}

func TestProfileDeclaresNoCLIOrCredentials(t *testing.T) {
	profile := Profile()

	// MiniMax reuses the codex binary; declaring a CLI here would add a
	// duplicate npm package and image label to every base image.
	if profile.CLI.Binary != "" || profile.CLI.PackageName != "" {
		t.Fatalf("profile should declare no CLI, got %#v", profile.CLI)
	}
	// The API key is injected as an environment variable, so there is nothing
	// to seed into or sync back out of a container.
	if !profile.Credentials.Empty() {
		t.Fatalf("profile should declare no credentials, got %#v", profile.Credentials)
	}
}

func TestCodexCLITracksCodexProfile(t *testing.T) {
	// The run-time CLI spec is read from the codex profile so the pinned
	// version can never drift between the two agents.
	if got, want := codexCLI(), codexagent.Profile().CLI; got != want {
		t.Fatalf("codexCLI() = %#v, want %#v", got, want)
	}
}

func TestProfileReturnsDefensiveCopies(t *testing.T) {
	first := Profile()
	first.Instructions.Path = "/changed"

	if second := Profile(); second.Instructions.Path == "/changed" {
		t.Fatal("Profile must return a defensive copy")
	}
}
