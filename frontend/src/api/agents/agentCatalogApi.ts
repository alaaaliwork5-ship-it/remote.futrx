import { requestJson } from "../apiRequest";
import type { AgentInfo } from "../../models/auth";
import { API_ROUTES } from "../../config/routes";

export const agentCatalogApi = {
  fetchAgents: (): Promise<AgentInfo[]> =>
    requestJson<{ agents: AgentInfo[] }>("GET", API_ROUTES.agents).then(
      (body) => body.agents
    ),
};
