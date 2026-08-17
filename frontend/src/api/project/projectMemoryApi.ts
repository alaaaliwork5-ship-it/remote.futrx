import { requestJson } from "../apiRequest";
import type { ProjectMemory } from "../../models/project";
import { API_ROUTES } from "../../config/routes";

export const projectMemoryApi = {
  getMemory: (id: string) =>
    requestJson<ProjectMemory>("GET", API_ROUTES.projects.memory(id)),

  setMemory: (id: string, content: string, enabled: boolean) =>
    requestJson<ProjectMemory>("PUT", API_ROUTES.projects.memory(id), {
      content,
      enabled,
    }),
};
