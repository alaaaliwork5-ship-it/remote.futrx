export interface LabeledOption<T extends string> {
  readonly value: T;
  readonly label: string;
}

export interface ModelOption extends LabeledOption<string> {
  readonly sub: string;
}

export const CHAT_PROVIDER_OPTIONS = [
  { value: "codex", label: "Codex", validInUserSettings: true },
  { value: "claude", label: "Claude", validInUserSettings: true },
  { value: "kimi", label: "Kimi", validInUserSettings: false },
  { value: "antigravity", label: "Antigravity", validInUserSettings: false },
  { value: "opencode", label: "OpenCode", validInUserSettings: true },
  { value: "freebuff", label: "Freebuff", validInUserSettings: false },
] as const;

export type ChatProvider = (typeof CHAT_PROVIDER_OPTIONS)[number]["value"];

export const CHAT_MODE_OPTIONS = [
  { value: "chat", label: "Chat" },
  { value: "plan", label: "Plan" },
  { value: "code", label: "Code" },
  { value: "review", label: "Review" },
  { value: "debug", label: "Debug" },
  { value: "full-auto", label: "Full auto" },
] as const;

export type ChatMode = (typeof CHAT_MODE_OPTIONS)[number]["value"];

// Reasoning-effort ladders differ per CLI (verified against each provider's
// --help / config validation). "Auto" ("") omits the flag so the CLI/server
// picks its own default.
//
// OpenCode forwards the selection as --variant (provider-specific reasoning
// effort) and silently ignores efforts the chosen model doesn't advertise, so
// the ladder spans both the OpenAI (none/xhigh) and Anthropic (max) families.
export const REASONING_EFFORT_OPTIONS = [
  { value: "", label: "Auto", providers: ["claude", "codex", "antigravity", "opencode"] },
  { value: "none", label: "None", providers: ["codex", "opencode"] },
  { value: "minimal", label: "Minimal", providers: ["codex"] },
  { value: "low", label: "Low", providers: ["claude", "codex", "antigravity", "opencode"] },
  { value: "medium", label: "Medium", providers: ["claude", "codex", "antigravity", "opencode"] },
  { value: "high", label: "High", providers: ["claude", "codex", "antigravity", "opencode"] },
  { value: "xhigh", label: "XHigh", providers: ["claude", "codex", "opencode"] },
  { value: "max", label: "Max", providers: ["claude", "codex", "opencode"] },
  { value: "ultra", label: "Ultra", providers: ["claude", "codex"] },
] as const;

export type ReasoningEffort = (typeof REASONING_EFFORT_OPTIONS)[number]["value"];

// Codex `service_tier` is the only headless speed lever across our providers.
// Values are model-gated; "Auto" ("") omits the flag entirely.
export const SERVICE_TIER_OPTIONS = [
  { value: "", label: "Auto", providers: ["codex"] },
  { value: "default", label: "Default", providers: ["codex"] },
  { value: "priority", label: "Priority", providers: ["codex"] },
  { value: "fast", label: "Fast", providers: ["codex"] },
] as const;

export type ServiceTier = (typeof SERVICE_TIER_OPTIONS)[number]["value"];

const MODEL_OPTIONS_BY_PROVIDER = {
  claude: [
    { value: "", label: "Auto", sub: "server default" },
    { value: "fable", label: "Fable", sub: "most capable" },
    { value: "opus", label: "Opus", sub: "deepest reasoning" },
    { value: "sonnet", label: "Sonnet", sub: "balanced" },
    { value: "haiku", label: "Haiku", sub: "fast" },
  ],
  codex: [
    { value: "", label: "Auto", sub: "codex default" },
    { value: "gpt-5.6-sol", label: "GPT-5.6 Sol", sub: "flagship preview" },
    { value: "gpt-5.5", label: "GPT-5.5", sub: "frontier coding" },
    { value: "gpt-5.4", label: "GPT-5.4", sub: "strong everyday coding" },
    { value: "gpt-5.4-mini", label: "GPT-5.4 Mini", sub: "fast" },
    { value: "gpt-5.3-codex", label: "GPT-5.3 Codex", sub: "coding optimized" },
  ],
  kimi: [{ value: "", label: "Auto", sub: "kimi default" }],
  // agy picks its own Gemini engine; the CLI accepts --model but publishes no
  // stable headless id list, so Auto is the only safe catalog entry.
  antigravity: [{ value: "", label: "Auto", sub: "antigravity default" }],
  // opencode routes provider/model IDs straight to the matching provider
  // (verified against the models.dev catalog it fetches). Anthropic models use
  // the ANTHROPIC_API_KEY project secret; OpenAI models use OPENAI_API_KEY.
  opencode: [
    { value: "", label: "Auto", sub: "opencode default" },
    { value: "anthropic/claude-opus-4-5", label: "Claude Opus 4.5", sub: "anthropic deep reasoning" },
    { value: "anthropic/claude-sonnet-4-5", label: "Claude Sonnet 4.5", sub: "anthropic balanced" },
    { value: "anthropic/claude-haiku-4-5", label: "Claude Haiku 4.5", sub: "anthropic fast" },
    { value: "openai/gpt-5.4", label: "GPT-5.4", sub: "openai strong coding" },
    { value: "openai/gpt-5.4-mini", label: "GPT-5.4 Mini", sub: "openai fast" },
  ],
  // freebuff is interactive-only; the model is chosen inside its TUI.
  freebuff: [{ value: "", label: "Auto", sub: "freebuff default" }],
} as const satisfies Record<ChatProvider, readonly ModelOption[]>;

export function modelOptionsForProvider(provider?: ChatProvider): readonly ModelOption[] {
  if (provider === "codex") return MODEL_OPTIONS_BY_PROVIDER.codex;
  if (provider === "kimi") return MODEL_OPTIONS_BY_PROVIDER.kimi;
  if (provider === "antigravity") return MODEL_OPTIONS_BY_PROVIDER.antigravity;
  if (provider === "opencode") return MODEL_OPTIONS_BY_PROVIDER.opencode;
  if (provider === "freebuff") return MODEL_OPTIONS_BY_PROVIDER.freebuff;
  return MODEL_OPTIONS_BY_PROVIDER.claude;
}

export function reasoningEffortOptionsForProvider(
  provider?: ChatProvider,
): readonly LabeledOption<ReasoningEffort>[] {
  const effectiveProvider = provider ?? "codex";
  return REASONING_EFFORT_OPTIONS.filter((option) =>
    supportsProvider(option, effectiveProvider),
  );
}

export function serviceTierOptionsForProvider(
  provider?: ChatProvider,
): readonly LabeledOption<ServiceTier>[] {
  const effectiveProvider = provider ?? "codex";
  return SERVICE_TIER_OPTIONS.filter((option) =>
    supportsProvider(option, effectiveProvider),
  );
}

export function providerDisplayLabel(provider?: ChatProvider): string {
  return CHAT_PROVIDER_OPTIONS.find((option) => option.value === provider)?.label ?? "Claude";
}

export function modelDisplayLabel(model?: string, provider?: ChatProvider): string {
  if (!model) return "Auto";
  if (provider === "codex") {
    const match = MODEL_OPTIONS_BY_PROVIDER.codex.find((option) => option.value === model);
    return match?.label ?? model;
  }
  if (provider === "opencode") {
    const match = MODEL_OPTIONS_BY_PROVIDER.opencode.find((option) => option.value === model);
    return match?.label ?? model;
  }
  return matchingClaudeModel(model)?.label ?? model;
}

export function modelShortLabel(model?: string): string {
  if (!model) return "auto";
  // opencode model IDs are provider/model pairs (e.g. openai/gpt-5.4); the
  // sidebar shows the model part only.
  const openCode = MODEL_OPTIONS_BY_PROVIDER.opencode.find((option) => option.value === model);
  if (openCode) return openCode.value.split("/").pop() ?? openCode.value;
  return matchingClaudeModel(model)?.value ?? model;
}

function supportsProvider(
  option: { readonly providers: readonly ChatProvider[] },
  provider: ChatProvider,
): boolean {
  return option.providers.includes(provider);
}

function matchingClaudeModel(model: string): ModelOption | undefined {
  const lower = model.toLowerCase();
  return MODEL_OPTIONS_BY_PROVIDER.claude.find(
    (option) => option.value !== "" && lower.includes(option.value),
  );
}
