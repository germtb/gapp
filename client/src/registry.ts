import { Store, type DecodedRpc } from "./store";
import { ok } from "./result";

export class StoreRegistry {
  private stores = new Set<Store<any>>();

  register<S extends Store<any>>(store: S): S {
    this.stores.add(store);
    return store;
  }

  dispatchRpc(event: unknown): void {
    for (const store of this.stores) {
      const current = store.getState();
      const next = store.reduceRpc(current, event);
      if (next !== current) store.setState(next);
    }
  }

  dispatchSendRpc(event: unknown): void {
    for (const store of this.stores) {
      const current = store.getState();
      const next = store.reduceSendRpc(current, event);
      if (next !== current) store.setState(next);
    }
  }

  hydrate(decoded: DecodedRpc[]): void {
    for (const event of decoded) {
      this.dispatchRpc({
        method: event.method,
        request: event.request,
        result: ok(event.response),
      });
    }
  }
}
