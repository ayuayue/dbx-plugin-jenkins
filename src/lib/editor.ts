/** 编辑器缩进单位，与 Jenkinsfile 常见的两空格缩进保持一致。 */
export const INDENT_UNIT = "  ";

export interface EditResult {
  value: string;
  selectionStart: number;
  selectionEnd: number;
}

/** 行号列文本，与 textarea 的行数逐行对齐（末尾换行也算一行）。 */
export function gutterText(source: string): string {
  const lines: string[] = [];
  const count = source.split("\n").length;
  for (let i = 1; i <= count; i += 1) lines.push(String(i));
  return lines.join("\n");
}

function lineRanges(value: string): Array<[number, number]> {
  const ranges: Array<[number, number]> = [];
  let start = 0;
  for (let i = 0; i < value.length; i += 1) {
    if (value[i] === "\n") {
      ranges.push([start, i]);
      start = i + 1;
    }
  }
  ranges.push([start, value.length]);
  return ranges;
}

/** 计算按下 Tab（outdent=false）或 Shift+Tab（outdent=true）后的文本与选区。 */
export function indentSelection(
  value: string,
  start: number,
  end: number,
  outdent = false,
): EditResult {
  const from = Math.max(0, Math.min(start, end, value.length));
  const to = Math.max(0, Math.min(Math.max(start, end), value.length));

  if (from === to) {
    if (!outdent) {
      const caret = from + INDENT_UNIT.length;
      return {
        value: value.slice(0, from) + INDENT_UNIT + value.slice(from),
        selectionStart: caret,
        selectionEnd: caret,
      };
    }
    const lineStart = value.lastIndexOf("\n", from - 1) + 1;
    const head = value.slice(lineStart, from);
    const spaces = / +$/.exec(head);
    const cut = head.endsWith("\t") ? 1 : spaces ? Math.min(2, spaces[0].length) : 0;
    const caret = from - cut;
    return {
      value: value.slice(0, caret) + value.slice(from),
      selectionStart: caret,
      selectionEnd: caret,
    };
  }

  const hits = lineRanges(value).map(([ls, le]) => ls < to && le >= from);
  let out = "";
  let cursor = 0;
  let delta = 0;
  let firstLineStart = -1;
  let firstDelta = 0;

  lineRanges(value).forEach(([ls, le], index) => {
    if (!hits[index]) return;
    out += value.slice(cursor, ls);
    const line = value.slice(ls, le);
    if (outdent) {
      const indent = /^(?:\t| {1,2})/.exec(line);
      const cut = indent ? indent[0].length : 0;
      out += line.slice(cut);
      delta -= cut;
      if (firstLineStart === -1) {
        firstLineStart = ls;
        firstDelta = -cut;
      }
    } else {
      out += INDENT_UNIT + line;
      delta += INDENT_UNIT.length;
      if (firstLineStart === -1) {
        firstLineStart = ls;
        firstDelta = INDENT_UNIT.length;
      }
    }
    cursor = le;
  });
  out += value.slice(cursor);

  const selectionStart = from === firstLineStart ? from : from + firstDelta;
  return { value: out, selectionStart, selectionEnd: to + delta };
}
