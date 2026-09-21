import { describe, it, expect } from 'vitest';
import { computeLineCount, indentCode, outdentCode } from './code-utils';

describe('Given code-utils helper', () => {
  describe('When computing line counts', () => {
    it('Then empty string returns at least 1 line', () => {
      expect(computeLineCount('')).toBe(1);
    });

    it('Then single line string returns 1 line', () => {
      expect(computeLineCount('export default {};')).toBe(1);
    });

    it('Then multi-line string returns exact line count', () => {
      const code = 'export default {\n  fetch() {\n    return new Response();\n  }\n};';
      expect(computeLineCount(code)).toBe(5);
    });
  });

  describe('When indenting code with Tab', () => {
    it('Then it inserts 2 spaces at cursor position', () => {
      const initial = 'const x = 1;';
      const { updatedCode, newStart, newEnd } = indentCode(initial, 6, 6, 2);
      expect(updatedCode).toBe('const   x = 1;');
      expect(newStart).toBe(8);
      expect(newEnd).toBe(8);
    });

    it('Then it replaces selected text with 2 spaces', () => {
      const initial = 'const old = 1;';
      const { updatedCode, newStart, newEnd } = indentCode(initial, 6, 9, 2);
      expect(updatedCode).toBe('const    = 1;');
      expect(newStart).toBe(8);
      expect(newEnd).toBe(8);
    });
  });

  describe('When outdenting code with Shift+Tab', () => {
    it('Then it removes 2 leading spaces from the line', () => {
      const initial = '  const x = 1;';
      const { updatedCode, newStart, newEnd } = outdentCode(initial, 4, 4, 2);
      expect(updatedCode).toBe('const x = 1;');
      expect(newStart).toBe(2);
      expect(newEnd).toBe(2);
    });

    it('Then it leaves unindented lines untouched', () => {
      const initial = 'const x = 1;';
      const { updatedCode, newStart, newEnd } = outdentCode(initial, 4, 4, 2);
      expect(updatedCode).toBe('const x = 1;');
      expect(newStart).toBe(4);
      expect(newEnd).toBe(4);
    });
  });
});
