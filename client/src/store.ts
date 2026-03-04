import { createCallbackSet } from "./callbackSet";

export type DecodedRpc = {
  method: string;
  request: unknown;
  response: unknown;
};

export abstract class Store<State, RpcResult = unknown, Action = never, RpcRequest = unknown> {
  protected state: State;
  private listeners = createCallbackSet<State>();

  constructor(initialState: State) {
    this.state = initialState;
    this.setState = this.setState.bind(this);
    this.getState = this.getState.bind(this);
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

  // Dispatch a local action (for non-RPC state changes)
  dispatch(action: Action): void {
    const newState = this.reduceAction(this.state, action);
    if (newState !== this.state) {
      this.setState(newState);
    }
  }

  // Pure reducer for RPC outcomes (success or error) - returns new state
  reduceRpc(state: State, _event: RpcResult): State {
    return state;
  }

  // Pure reducer for "about to send RPC" - returns new state (optional)
  reduceSendRpc(state: State, _event: RpcRequest): State {
    return state;
  }

  // Pure reducer for local actions - returns new state (optional)
  reduceAction(state: State, _action: Action): State {
    return state;
  }
}
