import type { ComponentChildren } from "preact";
import { createContext } from "preact";
import { useContext } from "preact/hooks";
import { useAuth, type AuthState } from "../hooks/auth/useAuth";
import { useClaudeAuth, type ClaudeAuthState } from "../hooks/auth/useClaudeAuth";
import { useCodexAuth, type CodexAuthState } from "../hooks/auth/useCodexAuth";
import { useKimiAuth, type KimiAuthState } from "../hooks/auth/useKimiAuth";
import { useOpenCodeAuth, type OpenCodeAuthState } from "../hooks/auth/useOpenCodeAuth";

interface AuthContextValue {
  auth: AuthState;
  claudeAuth: ClaudeAuthState;
  codexAuth: CodexAuthState;
  kimiAuth: KimiAuthState;
  opencodeAuth: OpenCodeAuthState;
  appAuthOk: boolean;
  providerAuthChecked: boolean;
  providerAuthenticated: boolean;
  gateOpen: boolean;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ComponentChildren }) {
  const auth = useAuth();
  // A valid local-admin or invited-user session may proceed to provider setup.
  const appAuthOk = auth.authenticated && (auth.isRegistered || auth.isAdmin);
  const providerAuthEnabled = appAuthOk && auth.localAdminConfigured;
  const claudeAuth = useClaudeAuth(providerAuthEnabled);
  const codexAuth = useCodexAuth(providerAuthEnabled);
  const kimiAuth = useKimiAuth(providerAuthEnabled);
  const opencodeAuth = useOpenCodeAuth(providerAuthEnabled);
  const providerAuthChecked =
    claudeAuth.checked && codexAuth.checked && kimiAuth.checked && opencodeAuth.checked;
  const providerAuthenticated =
    claudeAuth.authenticated ||
    codexAuth.authenticated ||
    kimiAuth.authenticated ||
    opencodeAuth.authenticated;
  const gateOpen = providerAuthEnabled && providerAuthChecked && providerAuthenticated;

  return (
    <AuthContext.Provider
      value={{
        auth,
        claudeAuth,
        codexAuth,
        kimiAuth,
        opencodeAuth,
        appAuthOk,
        providerAuthChecked,
        providerAuthenticated,
        gateOpen,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuthContext(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuthContext must be used inside AuthProvider");
  return value;
}
