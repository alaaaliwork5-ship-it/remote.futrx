package approval

import "regexp"

// dangerPatterns is the authoritative policy for commands that warrant a human
// approval before an agent runs them. The list is deliberately conservative:
// it flags destructive filesystem and device operations, privilege escalation,
// and common data-loss vectors. The container-side gate script only embeds a
// minimal pre-filter for the same shapes; this package is the source of truth.
type dangerPattern struct {
	reason string
	re     *regexp.Regexp
}

var dangerPatterns = []dangerPattern{
	{"recursively deleting a filesystem root or critical directory", regexp.MustCompile(`(?i)\brm\s+-[a-z]*[rf][a-z]*\s+[^;|&\n]*\s*(?:/|~|/root|/home|/etc|/usr|/var|/opt|/\*|\.)`)},
	{"deleting a device or block filesystem", regexp.MustCompile(`(?i)\brm\s+-[a-z]*[rf][a-z]*\s+/dev/`)},
	{"creating or formatting a filesystem", regexp.MustCompile(`(?i)\b(?:mkfs(?:\.\w+)?|mke2fs|wipefs|shred|fdisk\s+/dev/|parted\s+/dev/)`)},
	{"writing directly to a block device", regexp.MustCompile(`(?i)\bdd\b[^|;]*\bof=/dev/(?:sd|hd|vd|nvme|mmcblk|loop)[a-z0-9]*`)},
	{"writing to a block device via redirection", regexp.MustCompile(`(?i)(?:^|[;&|]\s*)[^>|;]*>\s*/dev/(?:sd|hd|vd|nvme|mmcblk|loop)[a-z0-9]*`)},
	{"fork bomb", regexp.MustCompile(`:\(\)\s*\{`)},
	{"recursively changing permissions on critical directories", regexp.MustCompile(`(?i)\bchmod\s+-R\s+[0-7]{3,4}\s+(?:/|/root|/etc|/usr|/var|/home)`)},
	{"changing ownership of critical directories", regexp.MustCompile(`(?i)\bchown\s+-R\s+[^\s]+\s+(?:/|/root|/etc|/usr|/var|/home)`)},
	{"shutting down or rebooting the machine", regexp.MustCompile(`(?i)\b(?:shutdown|reboot|halt|poweroff)\b`)},
	{"overwriting authentication or sudo files", regexp.MustCompile(`(?i)(?:>\s*/etc/(?:passwd|shadow|sudoers)|tee\s+/etc/(?:passwd|shadow|sudoers))`)},
	{"piping a remote script into a shell", regexp.MustCompile(`(?i)\b(?:curl|wget)\b[^|;]*\|\s*(?:sudo\s+)?(?:sh|bash|zsh)\b`)},
	{"force-pushing over a git remote", regexp.MustCompile(`(?i)\bgit\s+push\s+(?:-[a-z]*[fF][a-z]*|--force(?:-with-lease)?)`)},
	{"recursively deleting with find", regexp.MustCompile(`(?i)\bfind\s+[^|;]*-delete\b`)},
	{"moving the filesystem root", regexp.MustCompile(`(?i)\bmv\s+/\s+[^\s]+`)},
	{"dropping or erasing a database", regexp.MustCompile(`(?i)\b(?:drop\s+database|drop\s+table|truncate\s+table)\b`)},
}

// Reason returns a human-readable reason when the command matches a dangerous
// pattern. The empty string means the command is safe and can run without
// approval.
func Reason(command string) string {
	for _, p := range dangerPatterns {
		if p.re.MatchString(command) {
			return p.reason
		}
	}
	return ""
}
