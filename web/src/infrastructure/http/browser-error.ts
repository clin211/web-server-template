import { requestIdSchema } from "../../shared/validation/common";

type PublicErrorBody = Readonly<{
  error?: Readonly<{ code?: string; message?: string }>;
}>;

export class BrowserRequestError extends Error {
  readonly code: string;
  readonly status: number;
  readonly requestId: string | null;

  constructor(code: string, status: number, message: string, requestId: string | null = null) {
    super(message);
    this.name = "BrowserRequestError";
    this.code = code;
    this.status = status;
    const parsedRequestId = requestId ? requestIdSchema.safeParse(requestId) : null;
    this.requestId = parsedRequestId?.success ? parsedRequestId.data : null;
  }
}

export async function toBrowserRequestError(response: Response): Promise<BrowserRequestError> {
  const body = (await response.json().catch(() => null)) as PublicErrorBody | null;
  return new BrowserRequestError(
    body?.error?.code ?? "request-failed",
    response.status,
    body?.error?.message ?? "请求暂时无法完成",
    response.headers.get("x-request-id"),
  );
}
