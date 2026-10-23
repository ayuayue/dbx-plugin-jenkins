import type { ChangeSetItem } from "../types";

/** 折叠时默认展示的提交条数。 */
export const CHANGE_PREVIEW_LIMIT = 8;

/**
 * 最新在前：按提交时间倒序。
 * 时间缺失的条目保持原相对顺序（Jenkins 返回的是旧→新），排序必须稳定，
 * 否则同一秒的提交会在每次刷新时乱跳。
 */
export function sortChanges(items: ChangeSetItem[]): ChangeSetItem[] {
  return items
    .map((item, index) => ({ item, index }))
    .sort((left, right) => {
      const a = left.item.timestamp ?? 0;
      const b = right.item.timestamp ?? 0;
      if (a > 0 && b > 0 && a !== b) return b - a;
      if (a > 0 !== b > 0) return a > 0 ? -1 : 1;
      return left.index - right.index;
    })
    .map(({ item }) => item);
}

/** 折叠视图：展开时全量返回；limit 非正数视为不限。 */
export function visibleChanges(
  items: ChangeSetItem[],
  expanded: boolean,
  limit: number,
): ChangeSetItem[] {
  if (expanded || limit <= 0) return items;
  return items.slice(0, limit);
}

/** 面板头摘要：提交数、作者数（去重、忽略空作者）、改动文件总数。 */
export function changeSummary(items: ChangeSetItem[]): {
  commits: number;
  authors: number;
  files: number;
} {
  const authors = new Set<string>();
  let files = 0;
  for (const item of items) {
    const author = (item.author ?? "").trim();
    if (author) authors.add(author);
    files += Math.max(0, item.fileCount ?? 0);
  }
  return { commits: items.length, authors: authors.size, files };
}

/** 提交正文（后端已剥掉首行）。空字符串表示没有正文。 */
export function changeBody(item: ChangeSetItem): string {
  return (item.body ?? "").trim();
}

/** 是否有可展开的正文。 */
export function hasBody(item: ChangeSetItem): boolean {
  return changeBody(item).length > 0;
}

/** 提交时间：同年 `10-22 17:03`，跨年 `2025-12-31`，缺失破折号。 */
export function formatChangeTime(
  ms: number,
  locale: "zh" | "en",
  now = Date.now(),
): string {
  if (!Number.isFinite(ms) || ms <= 0) return "—";

  const date = new Date(ms);
  const pad = (value: number) => String(value).padStart(2, "0");
  if (date.getFullYear() !== new Date(now).getFullYear()) {
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
  }

  const time = `${pad(date.getHours())}:${pad(date.getMinutes())}`;
  if (locale === "en") {
    const month = date.toLocaleString("en-US", { month: "short" });
    return `${month} ${date.getDate()}, ${time}`;
  }
  return `${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${time}`;
}
