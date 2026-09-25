import { describe, it, expect } from 'vitest';
import { z } from 'zod';
import { isOk, isErr, unwrapOr } from './result';
import {
  validateSchema,
  environmentVariableSchema,
  resourceBindingSchema,
  domainSchema,
  createAppSchema,
} from './validation';

describe('Given the validation utility with Zod and Result pattern', () => {
  describe('When validating data with validateSchema', () => {
    const simpleSchema = z.object({
      name: z.string().min(3),
      age: z.number().int().positive(),
    });

    it('Then valid data returns Ok with parsed value', () => {
      const result = validateSchema(simpleSchema, { name: 'alice', age: 30 });
      expect(isOk(result)).toBe(true);
      expect(unwrapOr(result, null)).toEqual({ name: 'alice', age: 30 });
    });

    it('Then invalid data returns Err containing ZodError without throwing', () => {
      const result = validateSchema(simpleSchema, { name: 'al', age: -5 });
      expect(isErr(result)).toBe(true);
      if (isErr(result)) {
        expect(result.error).toBeInstanceOf(z.ZodError);
        expect(result.error.issues.length).toBeGreaterThanOrEqual(2);
      }
    });
  });

  describe('When validating environment variables', () => {
    it('Then valid env var passes schema', () => {
      const result = validateSchema(environmentVariableSchema, {
        key: 'DATABASE_URL',
        value: 'postgres://localhost:5432/db',
        isSecret: true,
      });
      expect(isOk(result)).toBe(true);
    });

    it('Then invalid env var key fails schema', () => {
      const result = validateSchema(environmentVariableSchema, {
        key: 'invalid-key!',
        value: 'some_value',
      });
      expect(isErr(result)).toBe(true);
    });
  });

  describe('When validating resource bindings', () => {
    it('Then valid binding passes schema', () => {
      const result = validateSchema(resourceBindingSchema, {
        name: 'KV_STORAGE',
        type: 'kv',
        resourceId: 'kv-12345',
      });
      expect(isOk(result)).toBe(true);
    });

    it('Then unsupported binding type fails schema', () => {
      const result = validateSchema(resourceBindingSchema, {
        name: 'AI_BINDING',
        type: 'unsupported_type',
        resourceId: 'res-1',
      });
      expect(isErr(result)).toBe(true);
    });
  });

  describe('When validating custom domains', () => {
    it('Then valid hostname passes schema', () => {
      const result = validateSchema(domainSchema, {
        hostname: 'api.example.com',
        pathPrefix: '/v1',
      });
      expect(isOk(result)).toBe(true);
    });

    it('Then invalid hostname fails schema', () => {
      const result = validateSchema(domainSchema, {
        hostname: 'not a valid hostname!',
      });
      expect(isErr(result)).toBe(true);
    });
  });

  describe('When validating application creation form', () => {
    it('Then valid git application passes schema', () => {
      const result = validateSchema(createAppSchema, {
        name: 'my-worker-app',
        sourceType: 'git',
        gitRepo: 'https://github.com/org/repo',
        gitBranch: 'main',
        autoDeploy: true,
      });
      expect(isOk(result)).toBe(true);
    });

    it('Then git application without gitRepo fails schema', () => {
      const result = validateSchema(createAppSchema, {
        name: 'my-worker-app',
        sourceType: 'git',
        gitRepo: '',
      });
      expect(isErr(result)).toBe(true);
    });

    it('Then uppercase or invalid characters in app name fails schema', () => {
      const result = validateSchema(createAppSchema, {
        name: 'INVALID NAME!',
        sourceType: 'inline',
        inlineCode: 'export default { fetch() {} }',
      });
      expect(isErr(result)).toBe(true);
    });
  });
});
