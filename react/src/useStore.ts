import { useCallback, useEffect, useState } from "react";
import type { Store } from "@gapp/client";

export function useStore<State>(store: Store<State, any, any, any>): State;
export function useStore<State, Selection>(
  store: Store<State, any, any, any>,
  selector?: (state: State) => Selection
): Selection;
export function useStore<State, Selection>(
  store: Store<State, any, any, any>,
  selector?: (state: State) => Selection,
  dependencies?: React.DependencyList
): Selection;

export function useStore<State, Selection>(
  store: Store<State, any, any, any>,
  selector?: (state: State) => Selection,
  dependencies?: React.DependencyList
): Selection {
  const getSelection = useCallback(
    (state: State) => (selector ? selector(state) : state),
    dependencies ?? []
  );

  const [selection, setSelection] = useState(() =>
    getSelection(store.getState())
  );

  useEffect(() => {
    return store.subscribe((state) => setSelection(getSelection(state)));
  }, [getSelection]);

  return selection as Selection;
}
