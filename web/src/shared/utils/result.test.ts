import { describe, it, expect } from 'vitest';
import {
  ok,
  err,
  isOk,
  isErr,
  map,
  mapErr,
  flatMap,
  unwrapOr,
  match,
  fromThrowable,
  fromPromise,
} from './result';

describe('Given the functional Result / Either type primitive', () => {
  describe('When creating Ok results', () => {
    it('Then isOk returns true and isErr returns false', () => {
      const res = ok(42);
      expect(isOk(res)).toBe(true);
      expect(isErr(res)).toBe(false);
      if (isOk(res)) {
        expect(res.value).toBe(42);
      }
    });

    it('Then map applies the transform to the inner value', () => {
      const res = ok('cubit');
      const mapped = map(res, (s) => s.toUpperCase());
      expect(isOk(mapped)).toBe(true);
      expect(unwrapOr(mapped, '')).toBe('CUBIT');
    });

    it('Then flatMap chains into a new Result', () => {
      const res = ok(10);
      const chained = flatMap(res, (n) => (n > 5 ? ok(n * 2) : err('too small')));
      expect(isOk(chained)).toBe(true);
      expect(unwrapOr(chained, 0)).toBe(20);
    });

    it('Then unwrapOr returns the inner value', () => {
      const res = ok('success');
      expect(unwrapOr(res, 'fallback')).toBe('success');
    });

    it('Then match calls the ok handler', () => {
      const res = ok(100);
      const text = match(res, {
        ok: (v) => `Value: ${v}`,
        err: (e) => `Error: ${e}`,
      });
      expect(text).toBe('Value: 100');
    });
  });

  describe('When creating Err results', () => {
    it('Then isErr returns true and isOk returns false', () => {
      const res = err(new Error('something went wrong'));
      expect(isErr(res)).toBe(true);
      expect(isOk(res)).toBe(false);
      if (isErr(res)) {
        expect(res.error.message).toBe('something went wrong');
      }
    });

    it('Then map preserves the Err without calling the transform', () => {
      const res = err('network error');
      const mapped = map(res, (s: string) => s.length);
      expect(isErr(mapped)).toBe(true);
      if (isErr(mapped)) {
        expect(mapped.error).toBe('network error');
      }
    });

    it('Then mapErr transforms the error value', () => {
      const res = err(404);
      const mapped = mapErr(res, (code) => `Status code: ${code}`);
      expect(isErr(mapped)).toBe(true);
      if (isErr(mapped)) {
        expect(mapped.error).toBe('Status code: 404');
      }
    });

    it('Then flatMap does not call the chained function on Err', () => {
      const res = err('initial failure');
      const chained = flatMap(res, (n: number) => ok(n * 2));
      expect(isErr(chained)).toBe(true);
      if (isErr(chained)) {
        expect(chained.error).toBe('initial failure');
      }
    });

    it('Then unwrapOr returns the fallback value', () => {
      const res = err('failed');
      expect(unwrapOr(res, 'default-value')).toBe('default-value');
    });

    it('Then match calls the err handler', () => {
      const res = err('auth_failed');
      const text = match(res, {
        ok: (v) => `Value: ${v}`,
        err: (e) => `Error: ${e}`,
      });
      expect(text).toBe('Error: auth_failed');
    });
  });

  describe('When wrapping unsafe operations', () => {
    it('Then fromThrowable captures thrown exceptions as Err', () => {
      const safeParse = fromThrowable(
        () => JSON.parse('invalid json{'),
        (e) => (e instanceof Error ? e.message : 'Unknown error')
      );
      expect(isErr(safeParse)).toBe(true);
    });

    it('Then fromThrowable returns Ok when no exception is thrown', () => {
      const safeParse = fromThrowable(() => JSON.parse('{"foo": "bar"}'));
      expect(isOk(safeParse)).toBe(true);
      expect(unwrapOr(safeParse, null)).toEqual({ foo: 'bar' });
    });

    it('Then fromPromise resolves to Ok on successful promise', async () => {
      const res = await fromPromise(Promise.resolve(99));
      expect(isOk(res)).toBe(true);
      expect(unwrapOr(res, 0)).toBe(99);
    });

    it('Then fromPromise resolves to Err on rejected promise', async () => {
      const res = await fromPromise(
        Promise.reject(new Error('async failure')),
        (e) => (e instanceof Error ? e.message : String(e))
      );
      expect(isErr(res)).toBe(true);
      if (isErr(res)) {
        expect(res.error).toBe('async failure');
      }
    });
  });
});
