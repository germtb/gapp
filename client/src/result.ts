export interface Result<T, E = Error> {
  isOk(): this is Ok<T, E>;
  isErr(): this is Err<T, E>;
  map<U>(fn: (value: T) => U): Result<U, E>;
  unwrap(): T | never;
  unwrapOrDefault(defaultValue: T): T;
}

export function ok<E = any>(): Result<void, E>;
export function ok<T, E = any>(value: T): Result<T, E>;
export function ok<T, E = any>(value?: T): Result<T | void, E> {
  return new Ok<T | void, E>(value as T);
}

export function err<T = any>(): Result<T, void>;
export function err<T, E = any>(error: E): Result<T, E>;
export function err<T, E = any>(error?: E): Result<T, E | void> {
  return new Err<T, E | void>(error as E);
}

export class Ok<T, E> implements Result<T, E> {
  constructor(public value: T) {}

  isOk(): this is Ok<T, E> {
    return true;
  }

  isErr(): this is Err<T, E> {
    return false;
  }

  map<U>(fn: (value: T) => U): Result<U, E> {
    return new Ok<U, E>(fn(this.value));
  }

  unwrap(): T {
    return this.value;
  }

  unwrapOrDefault(_defaultValue: T): T {
    return this.value;
  }
}

export class Err<T, E> implements Result<T, E> {
  constructor(public error: E) {}

  isOk(): this is Ok<T, E> {
    return false;
  }

  isErr(): this is Err<T, E> {
    return true;
  }

  map<U>(_fn: (value: T) => U): Result<U, E> {
    return new Err<U, E>(this.error);
  }

  unwrap(): T {
    throw new Error(`Called unwrap on Err: ${this.error}`);
  }

  unwrapOrDefault(defaultValue: T): T {
    return defaultValue;
  }
}
