import { Store } from "@gapp/client";
import type { RpcResult } from "../rpcTypes";
import type { Item } from "../generated/service";

type ItemState = {
  items: Item[];
};

class ItemStore extends Store<ItemState> {
  constructor() {
    super({ items: [] });
  }

  reduceRpc(state: ItemState, event: RpcResult): ItemState {
    if (event.method === "GetItems" && event.result.isOk()) {
      return { ...state, items: event.result.unwrap().items };
    }
    if (event.method === "CreateItem" && event.result.isOk()) {
      return { ...state, items: [...state.items, event.result.unwrap().item!] };
    }
    return state;
  }
}

export const itemStore = new ItemStore();
