export interface AuthSession {
  authenticated: boolean;
  claimed: boolean;
  localAdminConfigured: boolean;
  googleOAuthEnabled: boolean;
  googleClientId: string;
  adminEmail: string;
  email: string;
  isAdmin: boolean;
  isRegistered: boolean;
}

export interface GoogleOAuthSettings {
  configured: boolean;
  clientId: string;
  redirectUrl: string;
}

// Catalog entry served by GET /api/agents. The frontend renders one auth card
// per entry, choosing the card shape from authMethod so adding an agent is a
// backend-only change.
export interface AgentInfo {
  id: string;
  name: string;
  description: string;
  authMethod: "code" | "device" | "apikey" | "terminal";
  authAvailable: boolean;
  authenticated: boolean;
}

export type AgentAuthMethod = AgentInfo["authMethod"];

// Generic device-grant login state. Structurally identical across the device
// providers (Codex, Kimi, OpenCode); each provider's status payload is a
// superset of this common shape.
export interface AgentDeviceLogin {
  active: boolean;
  verificationUri?: string;
  userCode?: string;
  startedAt?: number;
  expiresAt?: number;
  completed?: boolean;
  error?: string;
}

// Common denominator of every provider auth-status payload, used by the
// catalog-driven settings cards.
export interface AgentAuthStatus {
  authenticated: boolean;
  authMode?: string;
  usesApiKey?: boolean;
  deviceLogin?: AgentDeviceLogin;
  login?: ClaudeLoginState;
}

export interface ClaudeAuthStatus {
  authenticated: boolean;
  login?: ClaudeLoginState;
}

// Streamed handshake state, mirroring CodexDeviceLogin. Claude's CLI uses the
// authorization-code grant instead of a device grant: it emits an OAuth URL
// and expects a code pasted back, so there's an `authUrl` + `awaitingCode`
// rather than a `userCode` for the user to read off.
export interface ClaudeLoginState {
  active: boolean;
  authUrl?: string;
  awaitingCode?: boolean;
  startedAt?: number;
  completed?: boolean;
  error?: string;
}

export interface ClaudeLoginStart {
  url: string;
  resumed?: boolean;
}

export interface CodexAuthStatus {
  authenticated: boolean;
  authMode?: string;
  usesApiKey?: boolean;
  deviceLogin?: CodexDeviceLogin;
}

export interface CodexDeviceLogin {
  active: boolean;
  verificationUri?: string;
  userCode?: string;
  startedAt?: number;
  expiresAt?: number;
  completed?: boolean;
  error?: string;
}

export interface KimiAuthStatus {
  authenticated: boolean;
  deviceLogin?: KimiDeviceLogin;
}

export interface KimiDeviceLogin {
  active: boolean;
  verificationUri?: string;
  userCode?: string;
  startedAt?: number;
  expiresAt?: number;
  completed?: boolean;
  error?: string;
}

export interface OpenCodeAuthStatus {
  authenticated: boolean;
  authMode?: string;
  usesApiKey?: boolean;
  deviceLogin?: OpenCodeDeviceLogin;
}

export interface OpenCodeDeviceLogin {
  active: boolean;
  verificationUri?: string;
  userCode?: string;
  startedAt?: number;
  expiresAt?: number;
  completed?: boolean;
  error?: string;
}

export type ClaudeLoginPhase =
  | "idle"
  | "starting"
  | "awaiting-code"
  | "submitting"
  | "done"
  | "error";
