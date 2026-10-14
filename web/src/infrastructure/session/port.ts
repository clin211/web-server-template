export type CredentialPolicy = { kind: "anonymous" } | { kind: "access" } | { kind: "refresh" };

export type TokenSet = {
  accessToken: string;
  refreshToken: string;
  accessExpiresAt: string;
  refreshExpiresAt: string;
};

export interface SessionPort {
  readAccess(request: Request): string | null;
  readRefresh(request: Request): string | null;
  commit(response: Response, tokens: TokenSet): Response;
  clear(response: Response): Response;
}
