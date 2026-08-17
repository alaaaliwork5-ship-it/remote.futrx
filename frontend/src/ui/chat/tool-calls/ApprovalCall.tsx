import { useState } from "preact/hooks";
import { approvalApi } from "../../../api/approvalApi";
import { AlertCircle, Check } from "../../primitives/icons";
import type { AssistantMessagePart } from "../../../models/chatMessage";

type ApprovalPart = Extract<AssistantMessagePart, { kind: "approval" }>;

export function ApprovalCall({ part }: { part: ApprovalPart }) {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function decide(decision: "allow" | "deny") {
    if (submitting || part.status !== "pending") return;
    setSubmitting(true);
    setError(null);
    try {
      await approvalApi.decide(part.approvalId, decision);
    } catch (cause) {
      setError((cause as Error).message);
      setSubmitting(false);
    }
  }

  const decided = part.status !== "pending";

  return (
    <div
      class={`my-2 rounded-lg border overflow-hidden text-sm ${
        part.status === "denied"
          ? "border-accent-red/50 bg-accent-red/5"
          : "border-accent-yellow/40 bg-accent-yellow/[0.04]"
      }`}
      role="group"
      aria-label="Shell command approval"
    >
      <div
        class={`px-3 py-2 text-[11px] font-medium flex items-center gap-1.5 border-b ${
          part.status === "denied"
            ? "bg-accent-red/10 text-accent-red border-accent-red/20"
            : "bg-accent-yellow/10 text-accent-yellow border-accent-yellow/20"
        }`}
      >
        {part.status === "denied" ? (
          <Check class="w-3.5 h-3.5" />
        ) : (
          <AlertCircle class="w-3.5 h-3.5" />
        )}
        <span>
          {part.status === "denied"
            ? "Command denied"
            : part.status === "allowed"
              ? "Command approved"
              : "Approval required"}
        </span>
        {decided && part.decidedBy && (
          <span class="ml-auto font-normal text-ink-300">by {part.decidedBy}</span>
        )}
      </div>

      <div class="p-3 space-y-3">
        {part.reason && (
          <p class="text-[13px] text-ink-200 leading-snug">
            This command may <span class="text-ink-100 font-medium">{part.reason}</span>.
          </p>
        )}
        <pre class="rounded-md bg-black/40 border border-white/10 px-3 py-2 text-[12.5px] font-mono text-ink-100 overflow-x-auto whitespace-pre-wrap break-words">
          {part.command}
        </pre>

        {!decided && (
          <div class="flex justify-end gap-2">
            <button
              type="button"
              disabled={submitting}
              onClick={() => decide("deny")}
              class="h-9 rounded-md px-4 text-sm font-medium text-ink-100 border border-white/15 hover:bg-white/5 disabled:opacity-60"
            >
              Deny
            </button>
            <button
              type="button"
              disabled={submitting}
              onClick={() => decide("allow")}
              class="h-9 rounded-md px-4 text-sm font-medium bg-white text-black hover:bg-white/90 disabled:opacity-60"
            >
              {submitting ? "Sending…" : "Approve and run"}
            </button>
          </div>
        )}

        {error && <p class="text-[13px] text-red-400">{error}</p>}
      </div>
    </div>
  );
}
