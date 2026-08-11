package minimax

import (
	"github.com/futrx-com/remote.futrx.com/internal/agent"
	codexagent "github.com/futrx-com/remote.futrx.com/internal/agent/codex"
	"github.com/futrx-com/remote.futrx.com/internal/agent/provisioning"
)

const (
	// containerMiniMaxHome is MiniMax's CODEX_HOME inside a project container.
	// It is deliberately NOT /root/.codex: pointing the same binary at a
	// separate home keeps MiniMax's config, session rollouts, and history fully
	// isolated from the ChatGPT-authenticated Codex agent, and guarantees the
	// two can never read each other's auth.json.
	containerMiniMaxHome = "/root/.minimax-codex"

	containerInstructions = containerMiniMaxHome + "/AGENTS.md"
	// Shared with the other agents on purpose: the workspace instructions
	// publisher batches every target that reports the same hash path into one
	// push, so reusing it writes MiniMax's AGENTS.md in the same pass.
	containerInstructionsHash = "/root/.claude/.agents-md.sha256"

	workspaceMiniMaxHome = "/workspace/.minimax-codex"
	containerSkills      = containerMiniMaxHome + "/skills"
)

var minimaxProfile = provisioning.Profile{
	ID: string(agent.ProviderMiniMax),
	// CLI is intentionally empty. MiniMax runs the Codex binary that the codex
	// profile already installs, and the image recipe skips profiles with no
	// CLI.Binary — declaring it again would add a duplicate npm package and
	// image label to every base image. Run-time availability is still
	// guaranteed: buildCmd ensures the codex CLI spec before exec.
	//
	// Credentials is intentionally empty too. The MiniMax API key is injected
	// into the run as an environment variable and never written to container
	// disk, so there is nothing to seed or sync back.
	Instructions: &provisioning.InstructionTarget{
		Path:     containerInstructions,
		HashPath: containerInstructionsHash,
	},
	WorkspaceSkills: &provisioning.WorkspaceSkills{
		WorkspaceHome: workspaceMiniMaxHome,
		HomeSkillsDir: containerSkills,
	},
}

// Profile returns MiniMax's container-facing policy. The returned value is a
// defensive copy so application wiring can compose profiles without mutating
// the provider's definition.
func Profile() provisioning.Profile {
	return minimaxProfile.Clone()
}

// codexCLI is the CLI spec MiniMax ensures at run time. It is read from the
// codex profile so the pinned version can never drift between the two agents.
func codexCLI() provisioning.CLISpec {
	return codexagent.Profile().CLI
}
