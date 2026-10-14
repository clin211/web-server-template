import { z } from "zod";
import { TransportError } from "../../infrastructure/http/errors";

export const apiSuccessSchema = z.object({
  code: z.literal(0),
  message: z.string(),
  data: z.unknown(),
});

export const apiFailureSchema = z.object({
  code: z.number(),
  reason: z.string(),
  message: z.string(),
});

export type ApiSuccess = z.infer<typeof apiSuccessSchema>;
export type ApiFailure = z.infer<typeof apiFailureSchema>;

export async function parseApiEnvelope(response: Response): Promise<ApiSuccess | ApiFailure> {
  let body: unknown;
  try {
    body = await response.json();
  } catch {
    throw new TransportError("invalid-envelope", "Upstream response was not valid JSON", {
      status: response.status,
      requestId: response.headers.get("x-request-id"),
    });
  }

  const success = apiSuccessSchema.safeParse(body);
  if (success.success && response.ok) return success.data;

  const failure = apiFailureSchema.safeParse(body);
  if (failure.success) return failure.data;

  throw new TransportError(
    "invalid-envelope",
    "Upstream response did not match the expected envelope",
    {
      status: response.status,
      requestId: response.headers.get("x-request-id"),
    },
  );
}
