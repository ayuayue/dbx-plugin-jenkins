/** 只保留最后 max 行，避免大日志拖垮渲染。 */
export function tailLines(text: string, max: number): string[] {
  if (!text) return [];
  const lines = text.split(/\r?\n/);
  if (lines.length > 0 && lines[lines.length - 1] === "") {
    lines.pop();
  }
  if (max > 0 && lines.length > max) {
    return lines.slice(-max);
  }
  return lines;
}

/**
 * 只保留日志尾部 maxChars 个字符，避免长构建把整份日志全留在内存里。
 * 界面本来就按 tailLines 截断显示，丢头部不影响可见内容；
 * 按行边界切，避免首行是半行。maxChars <= 0 表示不限制。
 */
export function trimLogTail(text: string, maxChars: number): string {
  if (maxChars <= 0 || text.length <= maxChars) return text;
  const tail = text.slice(-maxChars);
  const firstBreak = tail.indexOf("\n");
  return firstBreak < 0 ? tail : tail.slice(firstBreak + 1);
}

/** 统计所有行里 query 出现的次数（忽略大小写）。 */
export function countMatches(lines: string[], query: string): number {
  const needle = query.trim().toLowerCase();
  if (!needle) return 0;
  let total = 0;
  for (const line of lines) {
    const haystack = line.toLowerCase();
    let index = haystack.indexOf(needle);
    while (index >= 0) {
      total += 1;
      index = haystack.indexOf(needle, index + needle.length);
    }
  }
  return total;
}

export interface Segment {
  text: string;
  match: boolean;
}

/** 把一行拆成高亮片段，保留原始大小写。 */
export function highlightSegments(line: string, query: string): Segment[] {
  const needle = query.trim();
  if (!needle) return [{ text: line, match: false }];

  const haystack = line.toLowerCase();
  const target = needle.toLowerCase();
  const segments: Segment[] = [];
  let cursor = 0;

  while (cursor < line.length) {
    const found = haystack.indexOf(target, cursor);
    if (found < 0) {
      segments.push({ text: line.slice(cursor), match: false });
      break;
    }
    if (found > cursor) {
      segments.push({ text: line.slice(cursor, found), match: false });
    }
    segments.push({ text: line.slice(found, found + needle.length), match: true });
    cursor = found + needle.length;
  }

  return segments.length > 0 ? segments : [{ text: line, match: false }];
}
