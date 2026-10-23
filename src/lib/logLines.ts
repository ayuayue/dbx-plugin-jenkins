/** 日志行级别。info 只用于轻微弱化，不参与「仅问题」过滤。 */
export type LogLevel = "error" | "warn" | "info" | "plain";

/** ANSI 前景色的语义色名，渲染层再映射成具体色值。 */
export type AnsiTone = "red" | "green" | "yellow" | "blue" | "magenta" | "cyan" | "gray";

export interface StyledSpan {
  text: string;
  tone?: AnsiTone;
  bold?: boolean;
}

export interface LogRow {
  /** 当前缓冲区内的行号（1 起），不是 Jenkins 全量行号。 */
  number: number;
  /** 原始行文本，保留 ANSI 序列供渲染层上色。 */
  text: string;
  level: LogLevel;
}

/**
 * 匹配所有 ANSI 转义序列：
 * - SGR 颜色（ESC[...m）
 * - 其它 CSI（ESC[...K 等光标/擦除指令）
 * - OSC（ESC]...BEL 或 ESC]...ESC\，含超链接）
 * - 字符集选择（ESC(B）与单字符指令（ESC c）
 * SGR 放第一分支，保证颜色序列先于通用 CSI 命中。
 */
const ANSI_TOKEN =
  /\u001b\[[0-9;]*m|\u001b\[[0-9;?]*[ -/]*[@-~]|\u001b\][^\u0007\u001b]*(?:\u0007|\u001b\\)|\u001b[()#][0-9A-Za-z]|\u001b[@-_]/g;

/** 只剥掉转义序列，保留可见文本；日志本体来自 Jenkins，转义序列一律不显示。 */
export function stripAnsi(line: string): string {
  if (!line.includes("\u001b")) return line;
  return line.replace(ANSI_TOKEN, "");
}

const SGR_PREFIX = /^\u001b\[([0-9;]*)m$/;

/** 30-37 / 90-97 到语义色的映射；37/97 视为默认前景色，不额外着色。 */
const FOREGROUND: Record<number, AnsiTone | undefined> = {
  30: "gray",
  31: "red",
  32: "green",
  33: "yellow",
  34: "blue",
  35: "magenta",
  36: "cyan",
  37: undefined,
  90: "gray",
  91: "red",
  92: "green",
  93: "yellow",
  94: "blue",
  95: "magenta",
  96: "cyan",
  97: undefined,
};

/**
 * 把一行拆成带样式的片段。没有 ESC 时走快速路径，只产出一个纯文本片段；
 * 相邻同样式的片段合并，避免颜色切换产生大量空 span。
 */
export function parseAnsi(line: string): StyledSpan[] {
  if (!line) return [];
  if (!line.includes("\u001b")) return [{ text: line }];

  const spans: StyledSpan[] = [];
  let tone: AnsiTone | undefined;
  let bold = false;
  let cursor = 0;

  const push = (text: string): void => {
    if (!text) return;
    const last = spans[spans.length - 1];
    if (last && last.tone === tone && Boolean(last.bold) === bold) {
      last.text += text;
      return;
    }
    const span: StyledSpan = { text };
    if (tone) span.tone = tone;
    if (bold) span.bold = true;
    spans.push(span);
  };

  for (const match of line.matchAll(ANSI_TOKEN)) {
    const index = match.index ?? 0;
    push(line.slice(cursor, index));
    cursor = index + match[0].length;

    const sgr = SGR_PREFIX.exec(match[0]);
    if (!sgr) continue;
    const raw = sgr[1];
    const codes = raw === "" ? [0] : raw.split(";").map((part) => Number(part) || 0);
    for (const code of codes) {
      if (code === 0) {
        tone = undefined;
        bold = false;
      } else if (code === 1) {
        bold = true;
      } else if (code === 22) {
        bold = false;
      } else if (code === 39) {
        tone = undefined;
      } else if (code in FOREGROUND) {
        tone = FOREGROUND[code];
      }
    }
  }

  push(line.slice(cursor));
  return spans;
}

/** 错误行：大写级别词、构建失败、异常链、以及各工具的错误前缀。 */
const ERROR_RULES: RegExp[] = [
  /\b(?:ERROR|FATAL|SEVERE)\b/,
  /\b(?:BUILD FAILED|FAILURE|ABORTED)\b/,
  /Exception\b/,
  /Caused by:/,
  /\bnpm ERR!/,
  /\berror TS\d+/,
  /##\[error\]/i,
  /\berror:\s/i,
  /\b(?:FAILED|Failed)\s+to\b/,
];

const WARN_RULES: RegExp[] = [
  /\b(?:WARN|WARNING)\b/,
  /\bwarning:\s/i,
  /\bDEPRECATED\b/,
];

/** 只有行首的级别前缀才算 info，避免正文里的 "info" 被误伤。 */
const INFO_RULE = /^\s*\[?(?:INFO|DEBUG|TRACE)\]?(?:\s|:)/i;

/**
 * 行级别判定，保守优先：宁可少标不可错标。
 * `0 errors`、`errorRate` 这类包含 error 的普通词不命中（词边界 + 大写/冒号约束）。
 */
export function detectLevel(text: string): LogLevel {
  if (!text) return "plain";
  for (const rule of ERROR_RULES) {
    if (rule.test(text)) return "error";
  }
  for (const rule of WARN_RULES) {
    if (rule.test(text)) return "warn";
  }
  if (INFO_RULE.test(text)) return "info";
  return "plain";
}

/** 工具条徽标用的计数。 */
export function countLevels(lines: string[]): { error: number; warn: number } {
  let error = 0;
  let warn = 0;
  for (const line of lines) {
    const level = detectLevel(stripAnsi(line));
    if (level === "error") error += 1;
    else if (level === "warn") warn += 1;
  }
  return { error, warn };
}

/**
 * 过滤日志行。`problems` 只保留错误与警告，行号仍按原位置编号，
 * 跳号是预期行为（便于回到真实上下文）。
 */
export function filterLines(
  lines: string[],
  mode: "all" | "problems",
  firstLineNumber: number,
): LogRow[] {
  const rows: LogRow[] = [];
  lines.forEach((text, index) => {
    const level = detectLevel(stripAnsi(text));
    if (mode === "problems" && level !== "error" && level !== "warn") return;
    rows.push({ number: firstLineNumber + index, text, level });
  });
  return rows;
}

/**
 * 返回命中行的下标（0 起），用于上下跳转；与 countMatches 的「总处数」分工不同。
 * 一行命中多次只返回一次。
 */
export function matchRows(lines: string[], query: string): number[] {
  const needle = query.trim().toLowerCase();
  if (!needle) return [];
  const rows: number[] = [];
  lines.forEach((line, index) => {
    const text = stripAnsi(line).toLowerCase();
    if (text.includes(needle)) rows.push(index);
  });
  return rows;
}

/** 缓冲区只保留尾部时，第一条可见行的行号。 */
export function firstVisibleLine(total: number, visible: number): number {
  if (visible <= 0) return Math.max(1, total);
  return Math.max(1, total - visible + 1);
}
