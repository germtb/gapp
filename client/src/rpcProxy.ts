import { dispatchRpc, dispatchSendRpc } from "./store";
import { ok, err } from "./result";

/**
 * Creates a proxy around an RPC client that auto-dispatches to stores.
 * @param client The base RPC client to wrap
 * @param streamingMethods Methods that should not be wrapped (streaming methods)
 */
export function createRpcProxy<T extends object>(
  client: T,
  streamingMethods: Set<string> = new Set()
): T {
  return new Proxy(client, {
    get(target, prop: string) {
      const original = target[prop as keyof T];
      if (typeof original !== "function") {
        return original;
      }

      // Don't wrap streaming methods
      if (streamingMethods.has(prop)) {
        return original.bind(target);
      }

      // Wrap unary methods to auto-dispatch send + result (ok/err)
      return async (request: unknown) => {
        dispatchSendRpc({ method: prop, request });
        try {
          const response = await (original as Function).call(target, request);
          dispatchRpc({
            method: prop,
            request,
            result: ok(response),
          });
          return response;
        } catch (error) {
          dispatchRpc({
            method: prop,
            request,
            result: err(error as Error),
          });
          throw error;
        }
      };
    },
  }) as T;
}
