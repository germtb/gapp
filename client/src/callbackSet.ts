export function createCallbackSet<T>() {
  const set = new Set<(args: T) => void>();

  function add(callback: (args: T) => void) {
    const reference = (args: T) => callback(args);

    set.add(reference);

    return () => {
      set.delete(reference);
    };
  }

  function call(args: T) {
    // We need to do a copy because if any thing adds to the set during the call, it would cause an infinite loop
    const copy = Array.from(set);
    copy.forEach((callback) => callback(args));
  }

  function clear() {
    set.clear();
  }

  return { add, call, clear };
}
