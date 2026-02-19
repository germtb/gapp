export const RpcErrorCode = {
  VALIDATION_ERROR: "VALIDATION_ERROR",
  NOT_FOUND: "NOT_FOUND",
  ALREADY_EXISTS: "ALREADY_EXISTS",
  UNAUTHENTICATED: "UNAUTHENTICATED",
  PERMISSION_DENIED: "PERMISSION_DENIED",
  RATE_LIMITED: "RATE_LIMITED",
  INTERNAL: "INTERNAL",
} as const;

export type RpcErrorCodeType = (typeof RpcErrorCode)[keyof typeof RpcErrorCode];

export class RpcError extends Error {
  code: string;
  details: Record<string, string>;
  httpStatus: number;

  constructor(
    code: string,
    message: string,
    httpStatus: number,
    details?: Record<string, string>
  ) {
    super(message);
    this.name = "RpcError";
    this.code = code;
    this.details = details ?? {};
    this.httpStatus = httpStatus;
  }

  is(code: string): boolean {
    return this.code === code;
  }
}

export async function parseRpcError(
  res: Response
): Promise<RpcError> {
  const contentType = res.headers.get("Content-Type") ?? "";
  if (contentType.includes("application/json")) {
    try {
      const body = await res.json();
      return new RpcError(
        body.code ?? "UNKNOWN",
        body.message ?? res.statusText,
        res.status,
        body.details
      );
    } catch {
      // fall through to text parsing
    }
  }
  const text = await res.text();
  return new RpcError(
    "UNKNOWN",
    text.trim() || `HTTP error! status: ${res.status}`,
    res.status
  );
}
