import { Observable } from "rxjs";
import type { Result } from "./result";

/**
 * Utility types to extract RPC request/response types from service interfaces.
 * This avoids manually maintaining discriminated unions for each RPC method.
 */

// Extract all method names from a service interface (excluding non-function properties)
export type MethodNames<T> = {
  [K in keyof T]: T[K] extends (...args: any[]) => any ? K : never;
}[keyof T] &
  string;

// Extract request type from a method signature
export type RequestType<T> = T extends (req: infer R) => any ? R : never;

// Extract response type from a method signature (handles both Promise and Observable)
export type ResponseType<T> = T extends (req: any) => Promise<infer R>
  ? R
  : T extends (req: any) => Observable<infer R>
  ? R
  : never;

// Build discriminated union of { method, request } for all methods in a service
export type RpcRequestFromService<T> = {
  [K in MethodNames<T>]: { method: K; request: RequestType<T[K]> };
}[MethodNames<T>];

// Build discriminated union of { method, request, result } for all methods in a service
export type RpcResultFromService<T> = {
  [K in MethodNames<T>]: {
    method: K;
    request: RequestType<T[K]>;
    result: Result<ResponseType<T[K]>, Error>;
  };
}[MethodNames<T>];
