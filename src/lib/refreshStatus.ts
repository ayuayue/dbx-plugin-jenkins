import type { BuildSummary } from "../types";

/** 刷新徽章的四态：空闲 / 轮询中 / 请求中 / 已暂停。 */
export type RefreshTone = "idle" | "active" | "busy" | "paused";

/**
 * 传给视图组件的刷新 UI 状态。
 * 打包成一个对象，避免每个视图重复声明七八个 prop。
 */
export interface RefreshUi {
  mode: string | null;
  paused: boolean;
  inFlight: boolean;
  lastAt: number | null;
  intervalMs: number;
  /** 手动刷新请求进行中。 */
  refreshing: boolean;
  /** 手动刷新刚成功（1.5s 内）。 */
  refreshed: boolean;
}

/**
 * 徽章状态优先级：暂停 > 请求中 > 轮询中 > 空闲。
 * 暂停但已无轮询需求（例如构建刚结束）按空闲处理，避免残留提示。
 */
export function refreshBadge(
  mode: string | null,
  paused: boolean,
  inFlight: boolean,
): RefreshTone {
  if (!mode) return "idle";
  if (paused) return "paused";
  return inFlight ? "busy" : "active";
}

const pad2 = (value: number): string => String(value).padStart(2, "0");

/** 把时间戳格式化成 HH:mm:ss；无值时返回空串。 */
export function formatClockTime(timestamp: number | null | undefined): string {
  if (!timestamp) return "";
  const at = new Date(timestamp);
  return `${pad2(at.getHours())}:${pad2(at.getMinutes())}:${pad2(at.getSeconds())}`;
}

/**
 * 找出新增或状态发生变化的构建号，用于表格行高亮。
 * 首次加载（previous 为 null/undefined）不返回任何变化，避免整表闪烁。
 */
export function diffBuildRows(
  previous: BuildSummary[] | null | undefined,
  next: BuildSummary[] | null | undefined,
): number[] {
  if (previous == null || !next?.length) return [];
  const before = new Map<number, string>();
  for (const build of previous) before.set(build.number, build.status);
  const changed: number[] = [];
  for (const build of next) {
    const old = before.get(build.number);
    if (old === undefined || old !== build.status) changed.push(build.number);
  }
  return changed;
}
