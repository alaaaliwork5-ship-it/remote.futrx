import { requestJson } from "../../apiRequest";
import { subscribeToJsonMessages } from "../../../transport/jsonMessageSubscription";
import type { ApplicationPath } from "../../../types/transport";

interface ApiKeyAuthRoutes {
  readonly status: ApplicationPath;
  readonly submitKey: ApplicationPath;
  readonly clearKey: ApplicationPath;
  readonly statusUpdates: ApplicationPath;
}

// Shared client for providers authenticated by a static key the user pastes
// once, rather than by an interactive grant. The key is write-only: no endpoint
// returns it, and the status payload carries a masked hint only.
export class ApiKeyAuthApi<TStatus> {
  readonly #routes: ApiKeyAuthRoutes;

  constructor(routes: ApiKeyAuthRoutes) {
    this.#routes = routes;
  }

  readonly fetchStatus = (): Promise<TStatus> =>
    requestJson<TStatus>("GET", this.#routes.status);

  readonly submitKey = (key: string): Promise<{ success: boolean }> =>
    requestJson<{ success: boolean }>("POST", this.#routes.submitKey, { key });

  readonly clearKey = (): Promise<{ ok: boolean }> =>
    requestJson<{ ok: boolean }>("POST", this.#routes.clearKey, {});

  readonly subscribe = (
    onStatus: (status: TStatus) => void
  ): (() => void) =>
    subscribeToJsonMessages(this.#routes.statusUpdates, onStatus);
}
