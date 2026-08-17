import { useEffect, useState } from "preact/hooks";
import type { OpenCodeAuthStatus, OpenCodeDeviceLogin } from "../../../models/auth";
import { openCodeAuthApi } from "../../../api/agents/auth/openCodeAuthApi";

export interface OpenCodeAuthState {
  loading: boolean;
  checked: boolean;
  authenticated: boolean;
  usesApiKey: boolean;
  authMode?: string;
  deviceLogin?: OpenCodeDeviceLogin;
  starting: boolean;
  error: string | null;
  startDeviceLogin: () => Promise<void>;
}

export function useOpenCodeAuth(enabled: boolean): OpenCodeAuthState {
  const [loading, setLoading] = useState(false);
  const [checked, setChecked] = useState(false);
  const [authenticated, setAuthenticated] = useState(false);
  const [usesApiKey, setUsesApiKey] = useState(false);
  const [authMode, setAuthMode] = useState<string | undefined>(undefined);
  const [deviceLogin, setDeviceLogin] = useState<OpenCodeDeviceLogin | undefined>(undefined);
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function applyStatus(status: OpenCodeAuthStatus) {
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
      const state = await openCodeAuthApi.startDeviceLogin();
      setDeviceLogin(state);
    } catch (e) {
      setError((e as Error).message);
      throw e;
    } finally {
      setStarting(false);
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
    return openCodeAuthApi.subscribe(applyStatus);
  }, [enabled]);

  return {
    loading,
    checked,
    authenticated,
    usesApiKey,
    authMode,
    deviceLogin,
    starting,
    error,
    startDeviceLogin,
  };
}
