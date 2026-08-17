import { useEffect, useState } from "preact/hooks";
import type {
  AgentAuthStatus,
  AgentDeviceLogin,
} from "../../../models/auth";
import { agentAuthApi } from "../../../api/agents/auth/agentAuthApi";

export interface AgentAuthState {
  loading: boolean;
  // true once at least one status frame arrived. Mirrors the per-provider
  // hooks so the gating UI can distinguish "connecting" from "not configured".
  checked: boolean;
  authenticated: boolean;
  usesApiKey: boolean;
  authMode?: string;
  deviceLogin?: AgentDeviceLogin;
  starting: boolean;
  saving: boolean;
  error: string | null;
  startDeviceLogin: () => Promise<void>;
  saveKey: (provider: string, key: string) => Promise<void>;
}

// Generic host-auth status hook driven by the agent catalog. Subscribes to
// /ws/{provider}/auth-status like the per-provider hooks (useCodexAuth, ...),
// but works for any catalog-registered provider without provider-specific code.
export function useAgentAuth(agentId: string, enabled: boolean): AgentAuthState {
  const [loading, setLoading] = useState(false);
  const [checked, setChecked] = useState(false);
  const [authenticated, setAuthenticated] = useState(false);
  const [usesApiKey, setUsesApiKey] = useState(false);
  const [authMode, setAuthMode] = useState<string | undefined>(undefined);
  const [deviceLogin, setDeviceLogin] = useState<AgentDeviceLogin | undefined>(undefined);
  const [starting, setStarting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function applyStatus(status: AgentAuthStatus) {
    setAuthenticated(!!status.authenticated);
    setUsesApiKey(!!status.usesApiKey);
    setAuthMode(status.authMode);
    setDeviceLogin(status.deviceLogin);
    setError(null);
    setLoading(false);
    setChecked(true);
  }

  async function startDeviceLogin() {
    setStarting(true);
    setError(null);
    try {
      const state = await agentAuthApi(agentId).startDeviceLogin();
      setDeviceLogin(state);
    } catch (e) {
      setError((e as Error).message);
      throw e;
    } finally {
      setStarting(false);
    }
  }

  async function saveKey(provider: string, key: string) {
    setSaving(true);
    setError(null);
    try {
      await agentAuthApi(agentId).saveKey(provider, key);
    } catch (e) {
      setError((e as Error).message);
      throw e;
    } finally {
      setSaving(false);
    }
  }

  useEffect(() => {
    if (!enabled) {
      setLoading(false);
      setChecked(false);
      setAuthenticated(false);
      setUsesApiKey(false);
      setAuthMode(undefined);
      setDeviceLogin(undefined);
      setError(null);
      return;
    }

    setLoading(true);
    return agentAuthApi(agentId).subscribe(applyStatus);
  }, [agentId, enabled]);

  return {
    loading,
    checked,
    authenticated,
    usesApiKey,
    authMode,
    deviceLogin,
    starting,
    saving,
    error,
    startDeviceLogin,
    saveKey,
  };
}
