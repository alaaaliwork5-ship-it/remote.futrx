import { requestJson } from "../../apiRequest";
import { subscribeToJsonMessages } from "../../../transport/jsonMessageSubscription";
import type { ApplicationPath } from "../../../types/transport";
import type { AgentAuthStatus, AgentDeviceLogin } from "../../../models/auth";

export interface AgentLoginStart {
  url: string;
  resumed?: boolean;
}

// Generic per-agent auth client. Routes follow the shared host-side auth
// contract (`/api/{provider}/auth-status`, `/login/device`, `/login/start`,
// `/login/code`, `/login/cancel`, `/ws/{provider}/auth-status`) registered by
// the backend agent catalog, so one implementation serves every provider.
export function agentAuthApi(agentId: string) {
  const encodedId = encodeURIComponent(agentId);
  const prefix = `/api/${encodedId}`;
  const statusPath: ApplicationPath = `/ws/${encodedId}/auth-status`;

  return {
    fetchStatus: (): Promise<AgentAuthStatus> =>
      requestJson<AgentAuthStatus>("GET", `${prefix}/auth-status`),
    startDeviceLogin: (): Promise<AgentDeviceLogin> =>
      requestJson<AgentDeviceLogin>("POST", `${prefix}/login/device`, {}),
    startLogin: (): Promise<AgentLoginStart> =>
      requestJson<AgentLoginStart>("POST", `${prefix}/login/start`, {}),
    submitCode: (code: string): Promise<{ success: boolean }> =>
      requestJson<{ success: boolean }>("POST", `${prefix}/login/code`, { code }),
    cancelLogin: (): Promise<{ ok: boolean }> =>
      requestJson<{ ok: boolean }>("POST", `${prefix}/login/cancel`, {}),
    saveKey: (provider: string, key: string): Promise<{ ok: boolean }> =>
      requestJson<{ ok: boolean }>("POST", `${prefix}/login/key`, {
        provider,
        key,
      }),
    subscribe: (onStatus: (status: AgentAuthStatus) => void): (() => void) =>
      subscribeToJsonMessages(statusPath, onStatus),
  };
}
