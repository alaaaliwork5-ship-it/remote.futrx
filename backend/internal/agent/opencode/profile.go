package opencode

import (
	_ "embed"
	"os"
	"path/filepath"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
	"github.com/futrx-com/remote.futrx.com/internal/agent/provisioning"
)

const (
	// containerOpenCodeHome is where opencode keeps its data (auth.json,
	// session DB) inside a project container. The XDG data home defaults to
	// ~/.local/share; opencode stores under $XDG_DATA_HOME/opencode.
	containerOpenCodeHome = "/root/.local/share/opencode"

	// containerOpenCodeConfig is the XDG config home for opencode inside a
	// project container — where the shared AGENTS.md instructions land.
	containerOpenCodeConfig = "/root/.config/opencode"

	containerInstructionsPath = containerOpenCodeConfig + "/AGENTS.md"
	containerInstructionsHash = containerOpenCodeConfig + "/.agents-md.sha256"

	// browserMCPConfigPath is opencode's global config file inside a project
	// container. opencode 1.x loads MCP servers from mcpServers there; there is
	// no per-run --mcp-config flag like claude's, so the browser MCP entry is
	// pushed to the global config when the browser skill is selected.
	browserMCPConfigPath = containerOpenCodeConfig + "/opencode.json"
	browserMCPConfigHash = containerOpenCodeConfig + "/.mcp-opencode.sha256"

	missingCredentialsFormat = "opencode not authenticated — run `opencode auth login` on the host or `opencode login` in container %s"
)

//go:embed assets/mcp.json
var browserMCPConfig []byte

var opencodeProfile = provisioning.Profile{
	ID: string(agent.ProviderOpenCode),
	CLI: provisioning.CLISpec{
		Name:               "OpenCode",
		ImageLabel:         "opencode",
		Binary:             "opencode",
		PackageName:        "opencode-ai",
		Version:            provisioning.MustCLIVersion("OPENCODE_VERSION"),
		ReportVersion:      true,
		CheckVersion:       true,
		VerifyAfterInstall: true,
		InstallMode:        provisioning.InstallWithNPM,
		InstallTimeout:     8 * time.Minute,
		WaitTimeout:        5 * time.Minute,
	},
	Credentials: provisioning.CredentialSpec{
		Name: "opencode",
		Directory: &provisioning.CredentialDirectory{
			HostPath:                 hostOpenCodeHome(),
			ContainerPath:            containerOpenCodeHome,
			// Container paths are always Linux paths (the containers run
			// Ubuntu); avoid filepath so builds on any OS stay correct.
			ContainerDirs: []string{"/root/.local/share", containerOpenCodeHome},
			AllowContainerOnly:       true,
			MissingErrorFormat:       missingCredentialsFormat,
			SyncOnlyWhenHostHasFiles: false,
			SyncUnavailableIsNoop:    true,
		},
		SeedOnLaunch: false,
	},
	Instructions: &provisioning.InstructionTarget{
		Path:     containerInstructionsPath,
		HashPath: containerInstructionsHash,
	},
	BrowserMCPTemplates: []provisioning.TemplateFile{
		{
			Content:       append([]byte(nil), browserMCPConfig...),
			Path:          browserMCPConfigPath,
			HashPath:      browserMCPConfigHash,
			Mode:          "644",
			Directory:     containerOpenCodeConfig,
			DirectoryMode: "755",
		},
	},
}

// Profile returns OpenCode's container-facing policy. The returned value is a
// defensive copy so application wiring can compose profiles without mutating
// the provider's definition.
func Profile() provisioning.Profile {
	return opencodeProfile.Clone()
}

// hostOpenCodeHome mirrors opencode's XDG data resolution so the host-side
// auth/login, credential seeding, and status checks all agree on one path.
func hostOpenCodeHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "opencode")
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".local", "share", "opencode")
	}
	return containerOpenCodeHome
}
