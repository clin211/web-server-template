export type TransportErrorKind =
  | "network"
  | "timeout"
  | "unauthenticated"
  | "forbidden"
  | "invalid-envelope"
  | "upstream";

export class TransportError extends Error {
  readonly kind: TransportErrorKind;
  readonly status: number | null;
  readonly requestId: string | null;

  constructor(
    kind: TransportErrorKind,
    message: string,
    options: Readonly<{ status?: number | null; requestId?: string | null }> = {},
  ) {
    super(message);
    this.name = "TransportError";
    this.kind = kind;
    this.status = options.status ?? null;
    this.requestId = options.requestId ?? null;
  }
}
