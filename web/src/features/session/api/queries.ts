import { queryOptions } from "@tanstack/react-query";
import { fetchSession } from "./client";

export const sessionKeys = {
  all: ["session"] as const,
  current: () => [...sessionKeys.all, "current"] as const,
};

export const currentSessionQuery = () =>
  queryOptions({
    queryKey: sessionKeys.current(),
    queryFn: fetchSession,
    staleTime: 30_000,
    gcTime: 5 * 60_000,
    retry: false,
  });
