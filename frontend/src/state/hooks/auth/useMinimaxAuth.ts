import { useEffect, useState } from "preact/hooks";
import type { MiniMaxApiKeyState, MiniMaxAuthStatus } from "../../../models/auth";
import { minimaxAuthApi } from "../../../api/agents/auth/minimaxAuthApi";

export interface MiniMaxAuthState {
  loading: boolean;
  checked: boolean;
  authenticated: boolean;
  apiKey?: MiniMaxApiKeyState;
  saving: boolean;
  error: string | null;
  submitKey: (key: string) => Promise<void>;
  clearKey: () => Promise<void>;
}

export function useMinimaxAuth(enabled: boolean): MiniMaxAuthState {
  const [loading, setLoading] = useState(false);
  const [checked, setChecked] = useState(false);
  const [authenticated, setAuthenticated] = useState(false);
  const [apiKey, setApiKey] = useState<MiniMaxApiKeyState | undefined>(undefined);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function applyStatus(status: MiniMaxAuthStatus) {
    setAuthenticated(!!status.authenticated);
    setApiKey(status.apiKey);
    setError(null);
    setLoading(false);
    setChecked(true);
  }

  async function submitKey(key: string) {
    setSaving(true);
    setError(null);
    try {
      await minimaxAuthApi.submitKey(key);
      // The socket pushes the new status; no optimistic write is needed.
    } catch (e) {
      setError((e as Error).message);
      throw e;
    } finally {
      setSaving(false);
    }
  }

  async function clearKey() {
    setSaving(true);
    setError(null);
    try {
      await minimaxAuthApi.clearKey();
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
      setApiKey(undefined);
      setError(null);
      return;
    }

    setLoading(true);
    return minimaxAuthApi.subscribe(applyStatus);
  }, [enabled]);

  return {
    loading,
    checked,
    authenticated,
    apiKey,
    saving,
    error,
    submitKey,
    clearKey,
  };
}
