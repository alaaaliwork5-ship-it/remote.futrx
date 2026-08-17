import { useEffect, useRef, useState } from "preact/hooks";
import { useWorkspaceContext } from "../state/context/WorkspaceContext";

export function NewProjectDialog() {
  const workspace = useWorkspaceContext();
  const open = workspace.newProjectDialogOpen;
  const [name, setName] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!open) return;
    setName("");
    setError(null);
    setSubmitting(false);
    const frame = requestAnimationFrame(() => inputRef.current?.focus());
    return () => cancelAnimationFrame(frame);
  }, [open]);

  if (!open) return null;

  function close() {
    if (submitting) return;
    workspace.closeNewProjectDialog();
  }

  async function submit() {
    const trimmed = name.trim();
    if (!trimmed || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      await workspace.createProject(trimmed);
      workspace.closeNewProjectDialog();
    } catch (cause) {
      setError((cause as Error).message);
      setSubmitting(false);
    }
  }

  return (
    <div
      class="fixed inset-0 z-[70] bg-black/75 backdrop-blur-sm grid place-items-center p-4"
      role="dialog"
      aria-modal="true"
      aria-label="New project"
      onClick={close}
    >
      <div
        class="w-full max-w-sm rounded-xl border border-white/10 bg-[#17181d] p-5 shadow-2xl"
        onClick={(event) => event.stopPropagation()}
      >
        <h2 class="text-base font-medium text-ink-100">New project</h2>
        <p class="mt-1 text-[13px] text-ink-300">
          Each project gets its own sandboxed container. Agent CLIs install and
          run inside it.
        </p>
        <form
          class="mt-4 flex flex-col gap-3"
          onSubmit={(event) => {
            event.preventDefault();
            submit();
          }}
        >
          <input
            ref={inputRef}
            value={name}
            onInput={(event) =>
              setName((event.currentTarget as HTMLInputElement).value)
            }
            placeholder="Project name"
            disabled={submitting}
            autocomplete="off"
            class="h-10 w-full rounded-md border border-white/10 bg-white/5 px-3 text-sm text-ink-100 placeholder:text-ink-400 outline-none focus:border-white/25 disabled:opacity-60"
          />
          {error && (
            <p class="text-[13px] text-red-400" role="alert">
              {error}
            </p>
          )}
          <div class="flex justify-end gap-2">
            <button
              type="button"
              onClick={close}
              disabled={submitting}
              class="h-9 rounded-md px-3 text-sm text-ink-300 hover:bg-white/5 hover:text-ink-100 disabled:opacity-60"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={submitting || !name.trim()}
              class="h-9 rounded-md bg-white px-4 text-sm font-medium text-black hover:bg-white/90 disabled:opacity-50"
            >
              {submitting ? "Creating…" : "Create project"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
