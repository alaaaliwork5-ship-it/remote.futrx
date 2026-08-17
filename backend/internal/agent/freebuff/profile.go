// Package freebuff adapts Freebuff (the free, ad-supported coding agent) as a
// terminal-driven provider. The `freebuff` CLI (v0.0.x) is interactive-only:
// the only subcommand is `login`, there is no headless `run` command and no
// structured output stream. Remote therefore cannot execute prompts through it
// the way it drives Codex/Claude/OpenCode. Like Antigravity, Freebuff is a
// per-workspace terminal agent: it appears in the chat provider picker, is
// installed into project containers from the pinned npm package, and runs in
// the chat Terminal. A prompt sent with the Freebuff provider surfaces
// instructions instead of failing.
package freebuff

import (
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
	"github.com/futrx-com/remote.futrx.com/internal/agent/provisioning"
)

// terminalHint is the guidance emitted to the chat when a prompt is sent with
// the Freebuff provider, since the CLI has no headless execution path.
const terminalHint = "Freebuff is interactive-only — Remote can't run it headlessly (no `freebuff run` or JSON output in the pinned CLI yet). Open this chat's Terminal and run `freebuff`, then describe your task there."

var freebuffProfile = provisioning.Profile{
	ID: string(agent.ProviderFreebuff),
	CLI: provisioning.CLISpec{
		Name:               "Freebuff",
		ImageLabel:         "freebuff",
		Binary:             "freebuff",
		PackageName:        "freebuff",
		Version:            provisioning.MustCLIVersion("FREEBUFF_VERSION"),
		ReportVersion:      true,
		CheckVersion:       true,
		VerifyAfterInstall: true,
		InstallMode:        provisioning.InstallWithNPM,
		InstallTimeout:     8 * time.Minute,
		WaitTimeout:        5 * time.Minute,
	},
	// No credential policy: Freebuff is free (ad-supported) and needs no API
	// key or host-wide account. The CLI's optional in-TUI login is per
	// workspace and happens in the chat Terminal.
	Credentials: provisioning.CredentialSpec{Name: "freebuff"},
}

// Profile returns Freebuff's container-facing policy as a defensive copy.
func Profile() provisioning.Profile {
	return freebuffProfile.Clone()
}
