import { useEffect, useState } from "preact/hooks";
import type { ProjectMemory } from "../../../models/project";
import type { MemoryRecord } from "../../../state/projects/projectContainerRecords";
import { AlertCircle, Check, Loader } from "../../primitives/icons";
import { Loading } from "./ProjectContainerPrimitives";
import { formatUnixTime } from "./projectContainerFormat";

const MAX_MEMORY_BYTES = 32 * 1024;

export function ProjectMemorySection({
  record,
  onSave,
}: {
  record: MemoryRecord;
  onSave: (content: string, enabled: boolean) => Promise<ProjectMemory>;
}) {
  return (
    <>
      {record.error && (
        <div class="flex items-start gap-2.5 rounded-lg border border-accent-red/30 bg-accent-red/[0.08] px-3 py-2.5 text-[13px]">
          <AlertCircle class="w-4 h-4 mt-0.5 flex-none text-accent-red" />
          <div class="text-accent-red break-words">{record.error}</div>
        </div>
      )}
      {record.loading && !record.data ? (
        <Loading text="Loading project memory…" />
      ) : (
        <MemoryEditor initial={record.data} onSave={onSave} />
      )}
      <p class="text-[11.5px] text-ink-400 leading-relaxed">
        Project memory is a shared context document that is injected into the
        prompt of <em>every</em> agent run in this project — conventions,
        decisions, and state survive across chats and sessions. Keep it under
        ~32 KB and write it as plain text instructions the agents should follow.
      </p>
    </>
  );
}

function MemoryEditor({
  initial,
  onSave,
}: {
  initial?: ProjectMemory;
  onSave: (content: string, enabled: boolean) => Promise<ProjectMemory>;
}) {
  // An empty memory (nothing saved yet) defaults to enabled so the first save
  // is immediately injected; a saved-but-disabled memory stays disabled.
  const savedMemory = !!initial && initial.content !== "";
  const [content, setContent] = useState(initial?.content ?? "");
  const [enabled, setEnabled] = useState(savedMemory ? !!initial?.enabled : true);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [savedAt, setSavedAt] = useState<number | undefined>(initial?.updatedAt);

  useEffect(() => {
    const hadMemory = !!initial && initial.content !== "";
    setContent(initial?.content ?? "");
    setEnabled(hadMemory ? !!initial?.enabled : true);
    setSavedAt(initial?.updatedAt);
    setDirty(false);
    setErr(null);
  }, [initial]);

  const tooLarge = content.length > MAX_MEMORY_BYTES;

  const save = async () => {
    if (tooLarge) return;
    setSaving(true);
    setErr(null);
    try {
      const saved = await onSave(content, enabled);
      setSavedAt(saved.updatedAt);
      setDirty(false);
    } catch (error) {
      setErr((error as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div class="rounded-md border border-white/10 bg-white/[0.03] p-2.5 space-y-2.5">
      <div class="flex items-center gap-2 flex-wrap">
        <label class="flex items-center gap-2 text-[13px] text-ink-200 select-none">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(event) => {
              setEnabled((event.target as HTMLInputElement).checked);
              setDirty(true);
            }}
            class="accent-accent-blue w-4 h-4"
          />
          Inject into agent prompts
        </label>
        {savedAt !== undefined && (
          <span class="text-[11px] text-ink-400 ml-auto whitespace-nowrap">
            saved {formatUnixTime(savedAt)}
          </span>
        )}
      </div>
      <textarea
        value={content}
        onInput={(event) => {
          setContent((event.target as HTMLTextAreaElement).value);
          setDirty(true);
        }}
        rows={10}
        spellcheck={false}
        placeholder={
          "Example:\n- Use Go 1.25 and table-driven tests.\n- The deploy pipeline lives in .github/workflows — never edit it manually.\n- Prefer the project's existing component library over new UI dependencies."
        }
        class="w-full min-h-40 max-h-96 px-3 py-2.5 rounded border border-white/10 bg-black/30 text-[13px] font-mono text-ink-50 placeholder-ink-400 focus:outline-none focus:border-accent-blue/50 resize-y leading-[1.5] overflow-y-auto"
      />
      <div class="flex items-center gap-2 flex-wrap">
        <button
          type="button"
          onClick={save}
          disabled={!dirty || saving || tooLarge}
          class="h-9 px-3.5 rounded bg-accent-blue/80 hover:bg-accent-blue text-white text-[13px] font-medium disabled:opacity-50 inline-flex items-center gap-1.5"
        >
          {saving ? (
            <>
              <Loader class="w-3.5 h-3.5 animate-spin" /> Saving…
            </>
          ) : (
            <>
              <Check class="w-3.5 h-3.5" /> Save memory
            </>
          )}
        </button>
        {dirty && !saving && (
          <span class="text-[11.5px] text-ink-400">Unsaved changes</span>
        )}
        <span
          class={`text-[11.5px] ml-auto ${tooLarge ? "text-accent-red" : "text-ink-400"}`}
        >
          {content.length.toLocaleString()} / {MAX_MEMORY_BYTES.toLocaleString()} chars
          {tooLarge ? " — too large" : ""}
        </span>
      </div>
      {err && <div class="text-[11.5px] text-accent-red">{err}</div>}
    </div>
  );
}
