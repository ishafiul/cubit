export function computeLineCount(code: string): number {
  if (!code) return 1;
  return Math.max(code.split('\n').length, 1);
}

export function indentCode(
  code: string,
  start: number,
  end: number,
  tabSize = 2
): { updatedCode: string; newStart: number; newEnd: number } {
  const spaces = ' '.repeat(tabSize);
  const updatedCode = code.substring(0, start) + spaces + code.substring(end);
  return {
    updatedCode,
    newStart: start + tabSize,
    newEnd: start + tabSize,
  };
}

export function outdentCode(
  code: string,
  start: number,
  end: number,
  tabSize = 2
): { updatedCode: string; newStart: number; newEnd: number } {
  const spaces = ' '.repeat(tabSize);
  const before = code.substring(0, start);
  const lastNewline = before.lastIndexOf('\n');
  const lineStart = lastNewline === -1 ? 0 : lastNewline + 1;

  if (code.substring(lineStart, lineStart + tabSize) === spaces) {
    const updatedCode = code.substring(0, lineStart) + code.substring(lineStart + tabSize);
    return {
      updatedCode,
      newStart: Math.max(lineStart, start - tabSize),
      newEnd: Math.max(lineStart, end - tabSize),
    };
  }

  return { updatedCode: code, newStart: start, newEnd: end };
}
