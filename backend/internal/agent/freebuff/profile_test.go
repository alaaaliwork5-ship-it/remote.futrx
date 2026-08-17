package freebuff

import (
	"reflect"
	"testing"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/agent/provisioning"
)

func TestProfilePreservesFreebuffProvisioningPolicy(t *testing.T) {
	want := provisioning.Profile{
		ID: "freebuff",
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
		// Freebuff is free (ad-supported) — no credential policy at all.
		Credentials: provisioning.CredentialSpec{Name: "freebuff"},
	}

	if got := Profile(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Profile() = %#v, want %#v", got, want)
	}
	if !Profile().Credentials.Empty() {
		t.Fatalf("Freebuff profile must ship without a credential policy")
	}
}

func TestProfileReturnsDefensiveCopy(t *testing.T) {
	profile := Profile()
	profile.CLI.Name = "/changed"

	if got := Profile().CLI.Name; got != "Freebuff" {
		t.Fatalf("Profile() retained caller mutation: %q", got)
	}
}
