export interface Ok<T> {
  readonly ok: true;
  readonly value: T;
}

export interface Err<E> {
  readonly ok: false;
  readonly error: E;
}

export type Result<T, E = Error> = Ok<T> | Err<E>;

export function ok<T>(value: T): Ok<T> {
  return { ok: true, value };
}

export function err<E = Error>(error: E): Err<E> {
  return { ok: false, error };
}

export function isOk<T, E>(result: Result<T, E>): result is Ok<T> {
  return result.ok === true;
}

export function isErr<T, E>(result: Result<T, E>): result is Err<E> {
  return result.ok === false;
}

export function map<T, U, E>(result: Result<T, E>, fn: (value: T) => U): Result<U, E> {
  if (result.ok) {
    return ok(fn(result.value));
  }
  return result;
}

export function mapErr<T, E, F>(result: Result<T, E>, fn: (error: E) => F): Result<T, F> {
  if (!result.ok) {
    return err(fn(result.error));
  }
  return result;
}

export function flatMap<T, U, E>(
  result: Result<T, E>,
  fn: (value: T) => Result<U, E>
): Result<U, E> {
  if (result.ok) {
    return fn(result.value);
  }
  return result;
}

export function unwrapOr<T, E>(result: Result<T, E>, fallback: T): T {
  if (result.ok) {
    return result.value;
  }
  return fallback;
}

export function match<T, E, R>(
  result: Result<T, E>,
  matcher: {
    ok: (value: T) => R;
    err: (error: E) => R;
  }
): R {
  if (result.ok) {
    return matcher.ok(result.value);
  }
  return matcher.err(result.error);
}

export function fromThrowable<T, E = Error>(
  fn: () => T,
  mapErrFn?: (thrown: unknown) => E
): Result<T, E> {
  try {
    return ok(fn());
  } catch (caught) {
    if (mapErrFn) {
      return err(mapErrFn(caught));
    }
    return err(caught as E);
  }
}

export async function fromPromise<T, E = Error>(
  promise: Promise<T>,
  mapErrFn?: (thrown: unknown) => E
): Promise<Result<T, E>> {
  try {
    const value = await promise;
    return ok(value);
  } catch (caught) {
    if (mapErrFn) {
      return err(mapErrFn(caught));
    }
    return err(caught as E);
  }
}
