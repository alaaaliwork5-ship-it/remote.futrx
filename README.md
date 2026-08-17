<p align="center">
  <a href="https://remote.futrx.com/">
    <img src="docs.remote.futrx.com/static/brand/remote-futrx-on-dark.png" alt="Remote by FutrX" width="300">
  </a>
</p>

<h1 align="center">Give every AI project its own computer.</h1>

<p align="center">
  Run Codex, Claude Code, Kimi, Antigravity, OpenCode, and Freebuff in separate, always-on Linux workspaces on your own server.
  Use everything from one browser: chat, IDE, terminal, files, Git, live previews, and a shared browser.
</p>

<p align="center">
  <a href="https://remote.futrx.com/"><strong>Website</strong></a>
  ·
  <a href="https://docs.remote.futrx.com/"><strong>Documentation</strong></a>
  ·
  <a href="#quick-start"><strong>Install</strong></a>
  ·
  <a href="https://github.com/futrx-com/remote.futrx.com/issues"><strong>Roadmap</strong></a>
</p>

![Remote showing an AI conversation beside the application it built](docs/assets/readme/live-preview.webp)

## What is Remote?

Remote is an open-source, self-hosted home for AI coding agents.

Think of every project as its own server-side computer:

- It has a durable workspace, processes, ports, settings, and agent sessions.
- Codex, Claude Code, Kimi, Antigravity, OpenCode, and Freebuff can work in the same project without moving files between tools.
- The work keeps running on your server when you close your laptop.
- You can watch, review, edit, restart, or take over from any browser.

Remote is not another AI model. It gives the models you already use a complete place to work.

## New in this fork

This fork builds on upstream Remote with provider, safety, and deployment improvements:

**Dynamic agent catalog** — `GET /api/agents` serves the full provider list, and **Settings → Agents** renders every provider's auth card from that API (device-code, pasted-code, API-key, or terminal sign-in) instead of hardcoded cards.

**OpenCode provider** — OpenCode is a first-class provider:
- Model picker with Anthropic / OpenAI models and a reasoning-effort ladder mapped to opencode's `--variant` flag.
- Browser MCP wired through the container's global `opencode.json`, so the `browser` skill works exactly like Claude's.
- API-key sign-in (`opencode providers login`) stored in the container's `auth.json` and shared with project containers.

**Freebuff provider** — the free, ad-supported agent runs as a terminal-only provider: pick Freebuff in the chat, open the chat Terminal, and run `freebuff`. Its device-code login (`freebuff login`) is optional.

**Approval gate for Claude Code** — destructive shell commands (recursive deletes of filesystem roots, `mkfs`/`dd` on block devices, fork bombs, `curl|sh`, force-pushes, and more) are paused *before* they execute and presented as an **Approve and run / Deny** card in the chat. A `PreToolUse` hook enforces the host's danger policy even under `--dangerously-skip-permissions`; no decision within 8 minutes denies by default. Codex, Kimi, and OpenCode runs are not gated yet.

**Project memory** — every project has a shared context document (Settings → Memory, ≤ 32 KB) that is injected into the prompt of every agent run in that project, so conventions, decisions, and state survive across chats and sessions. Stored per project on the host with an enable/disable toggle.

**Deployment fixes** — the Docker build fails loudly on network errors, ships node 22 plus every agent CLI (claude, codex, kimi, opencode, freebuff), and the UI replaces `window.prompt` with an in-app project dialog (the browser-prompt API is blocked in embedded webviews).

## A quick tour

<table>
  <tr>
    <td width="50%">
      <img src="docs/assets/readme/create-project.webp" alt="Creating a new isolated project in Remote">
      <br>
      <strong>1. Create a project</strong><br>
      Remote prepares a separate Linux workspace with its own files, processes, ports, and agent homes.
    </td>
    <td width="50%">
      <img src="docs/assets/readme/parallel-agents.webp" alt="Multiple AI agents working in parallel in Remote">
      <br>
      <strong>2. Run agents in parallel</strong><br>
      Keep several chats moving while every agent works against the same project state.
    </td>
  </tr>
  <tr>
    <td width="50%">
      <img src="docs/assets/readme/browser-ide.webp" alt="Remote project opened in its browser IDE">
      <br>
      <strong>3. Inspect the real workspace</strong><br>
      Open the project in the browser IDE, terminal, file manager, or Git history.
    </td>
    <td width="50%">
      <img src="docs/assets/readme/agent-browser.webp" alt="Remote agent browser with human takeover">
      <br>
      <strong>4. Share the browser</strong><br>
      Watch the agent use a headed browser, then take control for sign-in or human judgment.
    </td>
  </tr>
</table>

See the continuous five-step product tour at [remote.futrx.com](https://remote.futrx.com/#product-tour).

## What you get

- **One project computer per project** — an unprivileged LXC container with durable files and agent homes.
- **Your choice of agent** — use Codex, Claude Code, Kimi, Antigravity, OpenCode, or Freebuff with provider-specific model and reasoning controls.
- **A complete development surface** — chat, browser IDE, root terminal, files, uploads, Git history, and reusable skills.
- **Live applications** — Remote finds listening ports, creates project URLs, adds HTTPS, and shows the app beside the conversation.
- **A browser agents and humans can share** — let an agent browse visually, watch it work, or take over the same session.
- **Scheduled work** — let a project chat run a one-time or recurring prompt later, even when your browser is closed.
- **Controls outside the workspace** — manage access, secrets, CPU, memory, lifecycle, and recovery from the Remote host.
- **Human approval for dangerous commands** — Claude Code pauses destructive shell commands until you approve or deny them in the chat.

## How it works

```mermaid
flowchart LR
    A["You<br>any browser"] --> B["Remote host<br>identity, routing, lifecycle"]
    B --> C["Project computer<br>one unprivileged LXC container"]
    C --> D["Codex · Claude · Kimi · Antigravity · OpenCode · Freebuff"]
    C --> E["IDE · terminal · Git · files"]
    C --> F["Browser · apps · HTTPS previews"]
```

The project computer is the capability boundary: agents can install tools, run servers, use Git, and browse inside it. The Remote host keeps authentication, routing, membership, and container lifecycle controls outside that boundary.

## Quick start

### What you need

- A fresh Ubuntu or Debian server
- Root or `sudo` access
- A hostname pointing at that server — one you own, or a free one
- A working SSH key
- Ports 80 and 443 open

> [!IMPORTANT]
> The installer disables SSH password login. Confirm that key-based SSH access works before you run it.

### 1. Point DNS to the server

Every project gets its own HTTPS address, so Remote needs a hostname with wildcard subdomains under it. Pick whichever case describes you — HTTPS is automatic in all three, with free Let's Encrypt certificates issued and renewed for you.

**If you already own a domain,** use a subdomain of it. For a base domain such as `remote.example.com`, create these records, all pointing at your server's IP address:

| DNS name | Purpose |
| --- | --- |
| `remote.example.com` | Remote web app |
| `code.remote.example.com` | Browser IDE |
| `*.code.remote.example.com` | Per-project browser IDEs |
| `*.dev.remote.example.com` | Per-project application previews |

**If you want a free hostname,** [DuckDNS](https://www.duckdns.org) is the quickest, because it resolves every subdomain automatically and there are no DNS records to create:

1. Sign in with GitHub or Google.
2. Add a name such as `yourname`. You now own `yourname.duckdns.org`.
3. Replace the pre-filled IP address with **your server's** address, then select **update ip**. The page fills in the address of the computer you are browsing from, which is usually not your server.

Then install using `yourname.duckdns.org` as the hostname.

[deSEC](https://desec.io) is a good alternative, run by a non-profit and less likely to be filtered on corporate networks. It is a full DNS host rather than a wildcard service, so create the four records from the table above under your `yourname.dedyn.io` name.

> [!NOTE]
> Free dynamic-DNS providers are community-run with no uptime guarantee, and some corporate networks block all of `*.duckdns.org` because of unrelated abuse elsewhere on it. If a preview link refuses to open at the office, that is usually why, and a domain you own avoids it.

**If you have neither,** a domain costs around $10 a year and gives you the shortest, most reliable URLs. Register one and follow the first case.

### 2. Install Remote

Connect to the server and run:

```bash
curl -fsSL https://remote.futrx.com/get | sudo bash -s -- remote.example.com
```

Replace `remote.example.com` with the hostname you set up above. The installer downloads Remote, installs its dependencies, builds the workspace image, starts the services, and enables HTTPS.

### 3. Create your first project

1. Visit `https://remote.example.com`.
2. Create the administrator account.
3. Open **Settings → Agents** and connect any of the six providers (Claude Code, Codex, Kimi, OpenCode via API key, or Antigravity/Freebuff from the chat Terminal).
4. Select **New project**.
5. Start a chat and describe what you want in normal language.

Remote will show the agent's progress. When the work is ready, review it in the chat, IDE, terminal, file manager, Git history, or live preview.

## Security in plain language

Remote is designed to reduce the blast radius of agent work, not to promise an air gap:

- Projects use separate unprivileged LXC containers, but they share the host kernel.
- The host administrator can access project data and controls.
- Secrets given to a project are readable by agents working in that project.
- Durable storage survives routine container replacement, but it is not a backup.

Before using Remote with valuable code or credentials, read the [threat model](docs/threat-model.md), [known limitations](docs/known-limitations.md), and [security policy](SECURITY.md).

## Updating

Run on the Remote server:

```bash
sudo bash /opt/remote.futrx/infra/update.sh
```

The updater rebuilds the app and project image while preserving project files and provider homes. Coordinate a maintenance window, or use `--skip-workspaces`, when agents are actively running. See [Deployment and operations](docs/04-operations/09-deployment-and-operations.md#update-flow) for details.

## Running this fork with Docker

The app itself runs fine in Docker while project computers use LXD on the host:

```bash
# From the repository root
docker build -t remotefutrx-backend:latest .

docker run -d --name remote-backend \
  -p 7682:7682 \
  -v "$(pwd)/.docker-data/data:/opt/remote.futrx/data" \
  -v "$(pwd)/.docker-data/projects:/var/lib/remote/projects" \
  remotefutrx-backend
```

The app is then at `http://localhost:7682`.

> [!IMPORTANT]
> Project computers are **LXD containers**, so the `lxc` CLI must be reachable **inside** the backend container and LXD must run on the host. Without it the app works (auth, settings, agent sign-in, chat) but **New project** fails with `lxc CLI not found on PATH - install LXD on the host first`. The supported path is a Linux server; on Windows you can run LXD inside WSL and expose it as a remote.

> [!NOTE]
> On Windows with Git Bash, prefix the `docker run` command with `MSYS2_ARG_CONV_EXCL="*"` so Git's path conversion does not rewrite `/opt/...` container paths into Windows paths.

Provider CLI sign-in happens inside the backend container, so the image ships node 22 and the `claude`, `codex`, `kimi`, `opencode`, and `freebuff` CLIs. The `freebuff` CLI downloads its ~50 MB binary on first run.

## Learn more

- [Documentation](https://docs.remote.futrx.com/) — operator and user guides
- [System architecture](ARCHITECTURE.md) — components, data flow, and trust boundaries
- [Project philosophy](docs/01-overview/00-philosophy.md) — why Remote treats each project as a computer
- [Contributing](CONTRIBUTING.md) — local development and contribution workflow
- [Issue tracker](https://github.com/alaaaliwork5-ship-it/remote.futrx/issues) — bugs, ideas, and roadmap

## License

Copyright © 2026 FutrX.

Remote is free software licensed under the [GNU Affero General Public License v3.0](LICENSE). You may self-host, modify, and redistribute it under the AGPL's terms. If you offer a modified version as a network service, the AGPL requires you to make the modified source available to its users.
