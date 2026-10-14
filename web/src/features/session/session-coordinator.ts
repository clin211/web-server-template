import {
  BrowserApiError,
  fetchSession as defaultFetchSession,
  refreshBrowserSession as defaultRefreshSession,
} from "./api/client";
import type { SessionData } from "./contracts";

type SessionCoordinatorDependencies = Readonly<{
  fetchSession?: () => Promise<SessionData>;
  refreshSession?: () => Promise<void>;
}>;

export type SessionCoordinator = Readonly<{
  requireSession: () => Promise<SessionData>;
  refreshOnce: () => Promise<void>;
}>;

export function createSessionCoordinator(
  dependencies: SessionCoordinatorDependencies = {},
): SessionCoordinator {
  const fetchSession = dependencies.fetchSession ?? defaultFetchSession;
  const refreshSession = dependencies.refreshSession ?? defaultRefreshSession;
  let refreshInFlight: Promise<void> | null = null;

  function refreshOnce(): Promise<void> {
    refreshInFlight ??= refreshSession().finally(() => {
      refreshInFlight = null;
    });
    return refreshInFlight;
  }

  async function requireSession(): Promise<SessionData> {
    try {
      return await fetchSession();
    } catch (error) {
      if (!(error instanceof BrowserApiError) || error.status !== 401) throw error;
      await refreshOnce();
      return fetchSession();
    }
  }

  return { requireSession, refreshOnce };
}
