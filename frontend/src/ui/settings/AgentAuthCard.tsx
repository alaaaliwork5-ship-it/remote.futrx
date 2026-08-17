import type { ComponentChildren } from "preact";
import { useEffect, useRef, useState } from "preact/hooks";
import type { AgentInfo } from "../../models/auth";
import { useAgentAuth } from "../../state/hooks/auth/useAgentAuth";
import { useAgentCodeLogin } from "../../state/hooks/auth/useAgentCodeLogin";
import { Check, ExternalLink, Key, Loader } from "../primitives/icons";

// Catalog-driven auth card. The backend decides the shape via authMethod
// ("code" / "device" / "terminal"), so adding an agent to the catalog renders
// the right card with no frontend changes.
export function AgentAuthCard({ agent }: { agent: AgentInfo }) {
  if (agent.authMethod === "terminal") return <TerminalAgentCard agent={agent} />;
  if (agent.authMethod === "code") return <CodeFlowAgentCard agent={agent} />;
  if (agent.authMethod === "apikey") return <APIKeyAgentCard agent={agent} />;
  return <DeviceFlowAgentCard agent={agent} />;
}

function CardHeader({
  agent,
  status,
}: {
  agent: AgentInfo;
  status: ComponentChildren;
}) {
  return (
    <div class="flex items-start gap-3">
      <div class="h-9 w-9 rounded-md bg-white/[0.06] border border-white/10 grid place-items-center flex-none text-ink-200">
        <Key class="w-4 h-4" />
      </div>
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2">
          <div class="text-[14px] font-semibold text-ink-100">
            {agent.name} authentication
          </div>
          {status}
        </div>
        <div class="text-[12px] text-ink-300 mt-1 leading-relaxed">
          {agent.description}
        </div>
      </div>
    </div>
  );
}

function SignedInBadge({ label }: { label: string }) {
  return (
    <span class="inline-flex items-center gap-1 text-[11px] text-accent-green">
      <Check class="w-3.5 h-3.5" /> {label}
    </span>
  );
}

function NotConfiguredBadge() {
  return <span class="text-[11px] text-ink-400">not configured</span>;
}

function DeviceFlowAgentCard({ agent }: { agent: AgentInfo }) {
  const auth = useAgentAuth(agent.id, true);
  const loginActive = !!auth.deviceLogin?.active;
  const expiresAt = auth.deviceLogin?.expiresAt
    ? new Date(auth.deviceLogin.expiresAt * 1000).toLocaleTimeString([], {
        hour: "numeric",
        minute: "2-digit",
      })
    : "";

  const status = auth.loading ? (
    <Loader class="w-3.5 h-3.5 text-ink-300 animate-spin" />
  ) : auth.authenticated ? (
    <SignedInBadge label={auth.usesApiKey ? "API keys configured" : "Signed in"} />
  ) : auth.usesApiKey ? (
    <span class="text-[11px] text-accent-red">API-key login detected</span>
  ) : (
    <NotConfiguredBadge />
  );

  return (
    <section class="rounded-md border border-white/10 bg-white/[0.03] p-3 space-y-3">
      <CardHeader agent={agent} status={status} />

      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => void auth.startDeviceLogin()}
          disabled={auth.starting || loginActive}
          class="h-10 px-3 rounded bg-accent-blue/80 hover:bg-accent-blue text-white text-[13px] font-medium disabled:opacity-50"
        >
          {auth.starting
            ? "Starting..."
            : loginActive
              ? "Login in progress"
              : auth.authenticated
                ? `Refresh ${agent.name} login`
                : `Sign in with ${agent.name}`}
        </button>
        {auth.loading && <Loader class="w-4 h-4 text-ink-300 animate-spin" />}
      </div>

      {auth.usesApiKey && !auth.authenticated && (
        <div class="text-[12px] text-accent-red bg-accent-red/[0.08] border border-accent-red/25 rounded px-2.5 py-2">
          {agent.name} is currently logged in with an API key. Complete the
          login above so prompts use subscription limits instead.
        </div>
      )}

      {loginActive && (
        <div class="rounded border border-accent-blue/25 bg-accent-blue/[0.08] p-3 space-y-2">
          <div class="text-[12px] text-ink-200">Open the link and confirm this code:</div>
          <div class="grid gap-2 sm:grid-cols-[1fr_auto]">
            <div class="font-mono text-[18px] tracking-wide text-ink-50 bg-black/30 border border-white/10 rounded px-3 py-2">
              {auth.deviceLogin?.userCode || "Waiting for code..."}
            </div>
            {auth.deviceLogin?.verificationUri && (
              <a
                href={auth.deviceLogin.verificationUri}
                target="_blank"
                rel="noreferrer"
                class="h-10 px-3 rounded bg-white/[0.08] hover:bg-white/[0.12] text-ink-100 text-[13px] font-medium inline-flex items-center justify-center gap-2"
              >
                <ExternalLink class="w-4 h-4" /> Open
              </a>
            )}
          </div>
          {expiresAt && (
            <div class="text-[11px] text-ink-400">Code expires around {expiresAt}.</div>
          )}
        </div>
      )}

      {(auth.deviceLogin?.error || auth.error) && (
        <div class="text-[12px] text-accent-red bg-accent-red/[0.08] border border-accent-red/25 rounded px-2.5 py-2 break-words">
          {auth.deviceLogin?.error || auth.error}
        </div>
      )}
    </section>
  );
}

function CodeFlowAgentCard({ agent }: { agent: AgentInfo }) {
  const auth = useAgentAuth(agent.id, true);
  const login = useAgentCodeLogin(agent.id, () => {});
  const codeRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (login.phase === "awaiting-code") {
      setTimeout(() => codeRef.current?.focus(), 50);
    }
  }, [login.phase]);

  const busy = login.phase === "starting" || login.phase === "submitting";
  const active = login.phase === "awaiting-code";
  const errorMessage = login.errorMessage || auth.error || "";

  const status = auth.loading ? (
    <Loader class="w-3.5 h-3.5 text-ink-300 animate-spin" />
  ) : auth.authenticated ? (
    <SignedInBadge label="Subscription signed in" />
  ) : (
    <NotConfiguredBadge />
  );

  return (
    <section class="rounded-md border border-white/10 bg-white/[0.03] p-3 space-y-3">
      <CardHeader agent={agent} status={status} />

      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => void login.startLogin()}
          disabled={busy || active}
          class="h-10 px-3 rounded bg-accent-blue/80 hover:bg-accent-blue text-white text-[13px] font-medium disabled:opacity-50"
        >
          {login.phase === "starting"
            ? "Starting..."
            : active
              ? "Login in progress"
              : auth.authenticated
                ? `Refresh ${agent.name} login`
                : `Sign in with ${agent.name}`}
        </button>
        {auth.loading && <Loader class="w-4 h-4 text-ink-300 animate-spin" />}
      </div>

      {active && (
        <div class="rounded border border-accent-blue/25 bg-accent-blue/[0.08] p-3 space-y-2">
          <div class="text-[12px] text-ink-200">
            Open the link, sign in, then paste the code shown back here:
          </div>
          <a
            href={login.authUrl}
            target="_blank"
            rel="noreferrer"
            class="block break-all text-accent-blue hover:underline font-mono text-[12px] bg-black/30 border border-white/10 rounded px-2.5 py-2"
          >
            <ExternalLink class="w-3.5 h-3.5 inline mr-1 align-[-2px]" />
            {login.authUrl}
          </a>
          <textarea
            ref={codeRef}
            value={login.code}
            onInput={(event) =>
              login.setCode((event.currentTarget as HTMLTextAreaElement).value)
            }
            onKeyDown={(event) => {
              if (event.key === "Enter" && !event.shiftKey && !event.isComposing) {
                event.preventDefault();
                void login.submitCode();
              }
            }}
            placeholder="Paste your code here"
            rows={2}
            autocomplete="off"
            autocapitalize="off"
            autocorrect="off"
            spellcheck={false}
            class="w-full resize-none rounded-md bg-black/30 border border-white/10 text-ink-100 placeholder:text-ink-300 px-3 py-2.5 font-mono text-[13px] focus:outline-none focus:border-accent-blue"
          />
          <div class="flex gap-2">
            <button
              type="button"
              onClick={() => void login.cancel()}
              class="px-3 h-10 text-[13px] text-ink-200 hover:text-ink-100 hover:bg-white/[0.08] rounded"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => void login.submitCode()}
              disabled={!login.code.trim()}
              class="flex-1 bg-accent-blue/80 hover:bg-accent-blue disabled:opacity-50 text-white text-[13px] font-medium rounded h-10"
            >
              Submit code
            </button>
          </div>
        </div>
      )}

      {login.phase === "submitting" && (
        <div class="flex items-center gap-2 text-[12px] text-ink-200">
          <Loader class="w-3.5 h-3.5 animate-spin" /> Finishing up
        </div>
      )}

      {errorMessage && !active && (
        <div class="text-[12px] text-accent-red bg-accent-red/[0.08] border border-accent-red/25 rounded px-2.5 py-2 break-words">
          {errorMessage}
        </div>
      )}
    </section>
  );
}

const apiKeyProviders = [
  { id: "anthropic", label: "Anthropic" },
  { id: "openai", label: "OpenAI" },
  { id: "google", label: "Google Gemini" },
];

// API-key credential card (OpenCode 1.18.x). The key is stored in the agent's
// credential file on the host and shared with project containers; no external
// login page is involved.
function APIKeyAgentCard({ agent }: { agent: AgentInfo }) {
  const auth = useAgentAuth(agent.id, true);
  const [provider, setProvider] = useState(apiKeyProviders[0].id);
  const [key, setKey] = useState("");
  const [saved, setSaved] = useState(false);

  async function save() {
    setSaved(false);
    try {
      await auth.saveKey(provider, key);
      setKey("");
      setSaved(true);
    } catch {
      // error surfaced via auth.error
    }
  }

  const status = auth.loading ? (
    <Loader class="w-3.5 h-3.5 text-ink-300 animate-spin" />
  ) : auth.authenticated ? (
    <SignedInBadge label="API keys configured" />
  ) : (
    <NotConfiguredBadge />
  );

  return (
    <section class="rounded-md border border-white/10 bg-white/[0.03] p-3 space-y-3">
      <CardHeader agent={agent} status={status} />

      <div class="grid gap-2 sm:grid-cols-[auto_1fr_auto]">
        <select
          value={provider}
          onChange={(event) =>
            setProvider((event.currentTarget as HTMLSelectElement).value)
          }
          class="h-10 rounded-md bg-black/30 border border-white/10 text-ink-100 text-[13px] px-2.5 focus:outline-none focus:border-accent-blue"
        >
          {apiKeyProviders.map((entry) => (
            <option value={entry.id}>{entry.label}</option>
          ))}
        </select>
        <input
          type="password"
          value={key}
          onInput={(event) =>
            setKey((event.currentTarget as HTMLInputElement).value)
          }
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              void save();
            }
          }}
          placeholder="Paste your API key"
          autocomplete="off"
          autocapitalize="off"
          autocorrect="off"
          spellcheck={false}
          class="w-full h-10 rounded-md bg-black/30 border border-white/10 text-ink-100 placeholder:text-ink-300 px-3 text-[13px] font-mono focus:outline-none focus:border-accent-blue"
        />
        <button
          type="button"
          onClick={() => void save()}
          disabled={auth.saving || !key.trim()}
          class="h-10 px-3 rounded bg-accent-blue/80 hover:bg-accent-blue text-white text-[13px] font-medium disabled:opacity-50"
        >
          {auth.saving ? "Saving..." : "Save key"}
        </button>
      </div>

      {saved && (
        <div class="text-[12px] text-accent-green">API key saved and shared with project containers.</div>
      )}

      {auth.error && (
        <div class="text-[12px] text-accent-red bg-accent-red/[0.08] border border-accent-red/25 rounded px-2.5 py-2 break-words">
          {auth.error}
        </div>
      )}
    </section>
  );
}

function TerminalAgentCard({ agent }: { agent: AgentInfo }) {
  return (
    <section class="rounded-md border border-white/10 bg-white/[0.03] p-3 space-y-3">
      <CardHeader
        agent={agent}
        status={<span class="text-[11px] text-ink-400">terminal sign-in</span>}
      />
    </section>
  );
}
