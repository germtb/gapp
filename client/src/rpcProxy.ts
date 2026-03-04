import type { StoreRegistry } from "./registry";
import { ok, err } from "./result";

/**
 * Creates a proxy around an RPC client that auto-dispatches to stores.
 * @param client The base RPC client to wrap
 * @param options.registry The store registry to dispatch events to
 * @param options.streamingMethods Methods that should not be wrapped (streaming methods)
 */
export function createRpcProxy<T extends object>(
  client: T,
  options: { registry: StoreRegistry; streamingMethods?: Set<string> },
): T {
  const { registry, streamingMethods = new Set() } = options;

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
        registry.dispatchSendRpc({ method: prop, request });
        try {
          const response = await (original as Function).call(target, request);
          registry.dispatchRpc({
            method: prop,
            request,
            result: ok(response),
          });
          return response;
        } catch (error) {
          registry.dispatchRpc({
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
