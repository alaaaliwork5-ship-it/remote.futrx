package opencode

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/agent/provisioning"
)

func TestProfilePreservesOpenCodeProvisioningPolicy(t *testing.T) {
	want := provisioning.Profile{
		ID: "opencode",
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
				ContainerPath:            "/root/.local/share/opencode",
				ContainerDirs:            []string{"/root/.local/share", "/root/.local/share/opencode"},
				AllowContainerOnly:       true,
				MissingErrorFormat:       "opencode not authenticated — run `opencode auth login` on the host or `opencode login` in container %s",
				SyncOnlyWhenHostHasFiles: false,
				SyncUnavailableIsNoop:    true,
			},
			SeedOnLaunch: false,
		},
		Instructions: &provisioning.InstructionTarget{
			Path:     "/root/.config/opencode/AGENTS.md",
			HashPath: "/root/.config/opencode/.agents-md.sha256",
		},
		BrowserMCPTemplates: []provisioning.TemplateFile{
			{
				Content:       append([]byte(nil), browserMCPConfig...),
				Path:          "/root/.config/opencode/opencode.json",
				HashPath:      "/root/.config/opencode/.mcp-opencode.sha256",
				Mode:          "644",
				Directory:     "/root/.config/opencode",
				DirectoryMode: "755",
			},
		},
	}

	if got := Profile(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Profile() = %#v, want %#v", got, want)
	}
}

func TestProfileBrowserMCPConfigPointsAtPlaywrightOverCDP(t *testing.T) {
	if len(browserMCPConfig) == 0 {
		t.Fatal("browser MCP config is empty")
	}

	var config struct {
		MCPServers []struct {
			Name    string   `json:"name"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(browserMCPConfig, &config); err != nil {
		t.Fatalf("parse browser MCP config: %v", err)
	}
	if len(config.MCPServers) != 1 {
		t.Fatalf("browser MCP servers = %d, want 1", len(config.MCPServers))
	}
	server := config.MCPServers[0]
	if server.Name != "browser" || server.Command != "npx" {
		t.Fatalf("browser MCP server = %#v, want npx @playwright/mcp", server)
	}
	if len(server.Args) == 0 || server.Args[0] != "@playwright/mcp" {
		t.Fatalf("browser MCP args = %v, want @playwright/mcp first", server.Args)
	}
	joined := strings.Join(server.Args, " ")
	if !strings.Contains(joined, "--cdp-endpoint") || !strings.Contains(joined, "127.0.0.1:9222") {
		t.Fatalf("browser MCP args %q do not attach to the shared Chrome core", joined)
	}
}

func TestProfileReturnsDefensiveCopy(t *testing.T) {
	profile := Profile()
	profile.Credentials.Directory.ContainerDirs[0] = "/changed"

	if got := Profile().Credentials.Directory.ContainerDirs[0]; got != "/root/.local/share" {
		t.Fatalf("Profile() retained caller mutation: %q", got)
	}
}
