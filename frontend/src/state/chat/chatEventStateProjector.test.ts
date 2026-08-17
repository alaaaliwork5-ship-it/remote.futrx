import assert from "node:assert/strict";
import test from "node:test";
import type { ChatEvent } from "../../models/chat.ts";
import { chatEventStateProjector } from "./chatEventStateProjector.ts";

test("projects chat events into the existing message and usage model", () => {
  const events: ChatEvent[] = [
    { type: "user", text: "hello", t: 1 },
    { type: "assistant_text", text: "hel", t: 2 },
    { type: "assistant_text", text: "lo", t: 3 },
    { type: "tool_use_start", id: "tool-1", name: "shell", input: { command: "pwd" }, t: 4 },
    { type: "tool_use_end", id: "tool-1", output: "/workspace", isError: false, t: 5 },
    { type: "complete", usage: { input_tokens: 3, output_tokens: 5 }, t: 6 },
  ];

  const state = chatEventStateProjector.fromEvents(events, {
    hasMore: false,
    lastSeq: 0,
  });

  assert.deepEqual(state.blocks, [
    { type: "user", text: "hello", t: 1 },
    {
      type: "assistant",
      parts: [
        { kind: "text", text: "hello" },
        {
          kind: "tool",
          id: "tool-1",
          name: "shell",
          input: { command: "pwd" },
          output: "/workspace",
          isError: false,
          status: "done",
        },
      ],
      t: 2,
      isComplete: true,
    },
  ]);

  assert.deepEqual(state.usageTotals, {
    inputTokens: 3,
    outputTokens: 5,
    cacheReadTokens: 0,
    cacheWriteTokens: 0,
  });
});

test("projects permission_request into an approval card and resolves it", () => {
  const events: ChatEvent[] = [
    {
      type: "permission_request",
      id: "approval-1",
      toolName: "Bash",
      input: { command: "rm -rf /" },
      data: { reason: "recursively deleting a filesystem root" },
      t: 10,
    },
  ];

  const pending = chatEventStateProjector.fromEvents(events, { hasMore: false, lastSeq: 0 });
  const assistant = pending.blocks[0];
  assert.equal(assistant.type, "assistant");
  if (assistant.type !== "assistant") return;
  assert.deepEqual(assistant.parts, [
    {
      kind: "approval",
      approvalId: "approval-1",
      toolName: "Bash",
      command: "rm -rf /",
      reason: "recursively deleting a filesystem root",
      status: "pending",
    },
  ]);

  const resolved = chatEventStateProjector.append(pending, [
    {
      type: "permission_request",
      id: "approval-1",
      toolName: "Bash",
      input: { command: "rm -rf /" },
      data: { decision: "deny", decidedBy: "admin@example.com" },
      t: 11,
    },
  ]);
  const resolvedAssistant = resolved.blocks[0];
  if (resolvedAssistant.type !== "assistant") return;
  assert.deepEqual(resolvedAssistant.parts, [
    {
      kind: "approval",
      approvalId: "approval-1",
      toolName: "Bash",
      command: "rm -rf /",
      reason: "recursively deleting a filesystem root",
      status: "denied",
      decidedBy: "admin@example.com",
    },
  ]);
});

test("drops permission_request without a command", () => {
  const state = chatEventStateProjector.fromEvents(
    [{ type: "permission_request", id: "approval-2", toolName: "Bash", input: {}, t: 1 }],
    { hasMore: false, lastSeq: 0 }
  );
  assert.equal(state.blocks.length, 0);
});
