import { createCsrfMiddleware, createMiddleware, createStart } from "@tanstack/react-start";
import type { RequestContext } from "./infrastructure/observability/request-context";
import { getRequestId, getTraceContext } from "./infrastructure/observability/request-context";

const requestContextMiddleware = createMiddleware({ type: "request" }).server(
  async ({ next, request }) => {
    const requestId = getRequestId(request);
    const context: RequestContext = {
      requestId,
      startedAt: Date.now(),
      traceId: getTraceContext(request)?.traceId ?? null,
    };
    const result = await next({ context });
    result.response.headers.set("x-request-id", requestId);
    return result;
  },
);

const serverFunctionCsrfMiddleware = createCsrfMiddleware({
  filter: (context) => context.handlerType === "serverFn",
});

export const startInstance = createStart(() => ({
  defaultSsr: false,
  requestMiddleware: [requestContextMiddleware, serverFunctionCsrfMiddleware],
}));

declare module "@tanstack/react-router" {
  interface Register {
    server: {
      requestContext: RequestContext;
    };
  }
}
