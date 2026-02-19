import { createCallbackSet } from "./callbackSet";
import { ok } from "./result";

export type DecodedRpc = {
  method: string;
  request: unknown;
  response: unknown;
};

export abstract class Store<State, Action = never> {
  protected state: State;
  private listeners = createCallbackSet<State>();
  public static all = new Set<Store<any, any>>();

  constructor(initialState: State) {
    this.state = initialState;
    this.setState = this.setState.bind(this);
    this.getState = this.getState.bind(this);
    Store.all.add(this);
  }

  getState(): State {
    return this.state;
  }

  setState(newState: State) {
    this.state = newState;
    this.listeners.call(this.state);
  }

  subscribe(callback: (newState: State) => void): () => void {
    callback(this.state);
    return this.listeners.add(callback);
  }

  cleanup() {
    Store.all.delete(this);
  }

  // Dispatch a local action (for non-RPC state changes)
  dispatch(action: Action): void {
    const newState = this.reduceAction(this.state, action);
    if (newState !== this.state) {
      this.setState(newState);
    }
  }

  // Pure reducer for RPC outcomes (success or error) - returns new state
  abstract reduceRpc(state: State, event: any): State;

  // Pure reducer for "about to send RPC" - returns new state (optional)
  reduceSendRpc(state: State, _event: any): State {
    return state;
  }

  // Pure reducer for local actions - returns new state (optional)
  reduceAction(state: State, _action: Action): State {
    return state;
  }
}

// Dispatch RPC outcome (success or error) to all stores
export function dispatchRpc(event: any): void {
  for (const store of Store.all) {
    const currentState = store.getState();
    const newState = store.reduceRpc(currentState, event);
    if (newState !== currentState) {
      store.setState(newState);
    }
  }
}

// Dispatch "about to send RPC" to all stores
export function dispatchSendRpc(event: any): void {
  for (const store of Store.all) {
    const currentState = store.getState();
    const newState = store.reduceSendRpc(currentState, event);
    if (newState !== currentState) {
      store.setState(newState);
    }
  }
}

// Hydrate from preloaded data (called once at app boot)
export async function dispatchPreloaded(
  decode: () => Promise<DecodedRpc[]>
): Promise<void> {
  const decoded = await decode();

  if (decoded.length === 0) {
    return;
  }

  // Dispatch each decoded request/response to stores (preloaded data is always successful)
  for (const event of decoded) {
    dispatchRpc({
      method: event.method,
      request: event.request,
      result: ok(event.response),
    });
  }
}
