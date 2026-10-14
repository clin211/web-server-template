import "@tanstack/react-start/server-only";
import { TransportError } from "./errors";

export async function readJsonBody(request: Request, maxBytes: number): Promise<unknown> {
  const contentType = request.headers.get("content-type")?.split(";", 1)[0].trim().toLowerCase();
  const contentLength = Number(request.headers.get("content-length") ?? "0");
  if (contentType !== "application/json") {
    throw new TransportError("invalid-envelope", "Request content type is not acceptable", {
      status: 415,
    });
  }
  if (Number.isFinite(contentLength) && contentLength > maxBytes) {
    throw new TransportError("invalid-envelope", "Request body is too large", { status: 413 });
  }

  const body = await request.text();
  if (new TextEncoder().encode(body).byteLength > maxBytes) {
    throw new TransportError("invalid-envelope", "Request body is too large", { status: 413 });
  }
  try {
    return JSON.parse(body) as unknown;
  } catch {
    throw new TransportError("invalid-envelope", "Request body is not valid JSON", {
      status: 400,
    });
  }
}
