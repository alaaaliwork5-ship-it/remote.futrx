import { useEffect, useState } from "preact/hooks";
import type { AgentInfo } from "../../../models/auth";
import { agentCatalogApi } from "../../../api/agents/agentCatalogApi";

export interface AgentCatalogState {
  agents: AgentInfo[] | null;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

// Fetches GET /api/agents. The catalog drives the dynamically rendered agent
// auth cards in Settings and the provider login screen.
export function useAgentCatalog(enabled: boolean): AgentCatalogState {
  const [agents, setAgents] = useState<AgentInfo[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    setLoading(true);
    setError(null);
    try {
      setAgents(await agentCatalogApi.fetchAgents());
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (!enabled) return;
    void refresh();
  }, [enabled]);

  return { agents, loading, error, refresh };
}
