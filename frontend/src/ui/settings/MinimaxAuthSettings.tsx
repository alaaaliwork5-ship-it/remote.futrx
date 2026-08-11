import { useState } from "preact/hooks";
import type { MiniMaxApiKeyState } from "../../models/auth";
import { Check, ExternalLink, Key, Loader } from "../primitives/icons";

const TOKEN_PLAN_URL = "https://platform.minimax.io/user-center/payment/token-plan";

// MiniMax has no interactive grant to drive: the user pastes a bearer token from
// its developer platform, so this panel is a write-only key field rather than a
// device-code handshake like Codex or Kimi.
export function MinimaxAuthSettings({
  authenticated,
  apiKey,
  loading,
  saving,
  error,
  onSubmitKey,
  onClearKey,
}: {
  authenticated: boolean;
  apiKey?: MiniMaxApiKeyState;
  loading: boolean;
  saving: boolean;
  error: string | null;
  onSubmitKey: (key: string) => Promise<void>;
  onClearKey: () => Promise<void>;
}) {
  const [draft, setDraft] = useState("");
  const [editing, setEditing] = useState(false);
  const showField = editing || !authenticated;

  async function submit(event: Event) {
    event.preventDefault();
    const key = draft.trim();
    if (!key || saving) return;
    try {
      await onSubmitKey(key);
      // Drop the plaintext as soon as it is stored so it does not linger in
      // component state or survive in a re-render.
      setDraft("");
      setEditing(false);
    } catch {
      // The hook surfaces the message through `error`.
    }
  }

  async function clear() {
    if (saving) return;
    try {
      await onClearKey();
      setDraft("");
      setEditing(false);
    } catch {
      // The hook surfaces the message through `error`.
    }
  }

  return (
    <section class="rounded-md border border-white/10 bg-white/[0.03] p-3 space-y-3">
      <div class="flex items-start gap-3">
        <div class="h-9 w-9 rounded-md bg-white/[0.06] border border-white/10 grid place-items-center flex-none text-ink-200">
          <Key class="w-4 h-4" />
        </div>
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2">
            <div class="text-[14px] font-semibold text-ink-100">MiniMax authentication</div>
            {loading ? (
              <Loader class="w-3.5 h-3.5 text-ink-300 animate-spin" />
            ) : authenticated ? (
              <span class="inline-flex items-center gap-1 text-[11px] text-accent-green">
                <Check class="w-3.5 h-3.5" /> API key stored{" "}
                {apiKey?.hint && <span class="font-mono text-ink-300">{apiKey.hint}</span>}
              </span>
            ) : (
              <span class="text-[11px] text-ink-400">not configured</span>
            )}
          </div>
          <div class="text-[12px] text-ink-300 mt-1 leading-relaxed">
            Runs MiniMax models through the Codex CLI against{" "}
            <span class="font-mono text-ink-100">api.minimax.io</span>, with its own session
            history kept separate from Codex. Billed against your MiniMax token plan.
          </div>
          <a
            href={TOKEN_PLAN_URL}
            target="_blank"
            rel="noreferrer"
            class="text-[12px] text-accent-blue hover:underline inline-flex items-center gap-1 mt-1"
          >
            <ExternalLink class="w-3.5 h-3.5" /> Get an API key
          </a>
        </div>
      </div>

      {showField ? (
        <form class="grid gap-2 sm:grid-cols-[1fr_auto]" onSubmit={submit}>
          <input
            type="password"
            value={draft}
            autocomplete="off"
            spellcheck={false}
            placeholder="Paste your MiniMax API key"
            onInput={(e) => setDraft((e.target as HTMLInputElement).value)}
            class="h-10 px-3 rounded bg-black/30 border border-white/10 text-ink-100 text-[13px] font-mono placeholder:text-ink-500 focus:outline-none focus:border-accent-blue/50"
          />
          <div class="flex items-center gap-2">
            <button
              type="submit"
              disabled={saving || !draft.trim()}
              class="h-10 px-3 rounded bg-accent-blue/80 hover:bg-accent-blue text-white text-[13px] font-medium disabled:opacity-50"
            >
              {saving ? "Saving..." : "Save key"}
            </button>
            {editing && (
              <button
                type="button"
                onClick={() => {
                  setDraft("");
                  setEditing(false);
                }}
                class="h-10 px-3 rounded bg-white/[0.08] hover:bg-white/[0.12] text-ink-100 text-[13px] font-medium"
              >
                Cancel
              </button>
            )}
          </div>
        </form>
      ) : (
        <div class="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => setEditing(true)}
            disabled={saving}
            class="h-10 px-3 rounded bg-white/[0.08] hover:bg-white/[0.12] text-ink-100 text-[13px] font-medium disabled:opacity-50"
          >
            Replace key
          </button>
          <button
            type="button"
            onClick={() => void clear()}
            disabled={saving}
            class="h-10 px-3 rounded bg-white/[0.08] hover:bg-white/[0.12] text-ink-100 text-[13px] font-medium disabled:opacity-50"
          >
            {saving ? "Removing..." : "Remove key"}
          </button>
          {loading && <Loader class="w-4 h-4 text-ink-300 animate-spin" />}
        </div>
      )}

      {(apiKey?.error || error) && (
        <div class="text-[12px] text-accent-red bg-accent-red/[0.08] border border-accent-red/25 rounded px-2.5 py-2 break-words">
          {apiKey?.error || error}
        </div>
      )}
    </section>
  );
}
