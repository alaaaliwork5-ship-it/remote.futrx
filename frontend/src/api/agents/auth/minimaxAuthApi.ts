import { ApiKeyAuthApi } from "./apiKeyAuthApi";
import type { MiniMaxAuthStatus } from "../../../models/auth";
import { API_ROUTES, WEB_SOCKET_ROUTES } from "../../../config/routes";

export const minimaxAuthApi = new ApiKeyAuthApi<MiniMaxAuthStatus>({
  status: API_ROUTES.minimaxAuth.status,
  submitKey: API_ROUTES.minimaxAuth.submitKey,
  clearKey: API_ROUTES.minimaxAuth.clearKey,
  statusUpdates: WEB_SOCKET_ROUTES.minimaxAuthStatus,
});
