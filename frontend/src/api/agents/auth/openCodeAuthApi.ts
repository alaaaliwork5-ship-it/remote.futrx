import { DeviceAuthApi } from "./deviceAuthApi";
import type { OpenCodeAuthStatus, OpenCodeDeviceLogin } from "../../../models/auth";
import { API_ROUTES, WEB_SOCKET_ROUTES } from "../../../config/routes";

export const openCodeAuthApi = new DeviceAuthApi<OpenCodeAuthStatus, OpenCodeDeviceLogin>({
  status: API_ROUTES.opencodeAuth.status,
  startDeviceLogin: API_ROUTES.opencodeAuth.startDeviceLogin,
  statusUpdates: WEB_SOCKET_ROUTES.opencodeAuthStatus,
});
