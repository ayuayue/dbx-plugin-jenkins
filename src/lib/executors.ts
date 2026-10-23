import type { ExecutorInfo } from "../types";
import { buildProgress, type BuildProgress } from "./progress";

/** 执行器四种状态：空闲 / 等待启动 / 构建中 / 收尾中。 */
export type ExecutorState = "idle" | "pending" | "building" | "settling";

/**
 * 节点显示名归一：控制器的 `displayName` 是空串，用传入的文案兜底。
 * 文案由调用方给（走 i18n），纯函数不内置中文/英文。
 */
export function executorNodeLabel(name: string, builtInLabel: string): string {
  const trimmed = (name ?? "").trim();
  return trimmed === "" ? builtInLabel : trimmed;
}

/**
 * 执行器状态。
 *
 * `settling` 是构建已结束但执行器还没释放的瞬间：Jenkins 会短暂保留
 * `currentExecutable`，此时不该显示「空闲」，否则状态会先跳空闲再跳回来。
 */
export function executorState(row: ExecutorInfo): ExecutorState {
  if (row.idle) return "idle";
  if (row.pending) return "pending";
  if (row.building) return "building";
  if (row.buildNumber > 0 || row.jobName !== "") return "settling";
  return "pending";
}

/** 执行器占用汇总：`!idle` 即被占用，空闲槽位也计入 total。 */
export function executorSummary(rows: ExecutorInfo[]): { busy: number; total: number } {
  let busy = 0;
  for (const row of rows) {
    if (!row.idle) busy++;
  }
  return { busy, total: rows.length };
}

/**
 * 执行器进度：复用构建列表的时长估算（服务端 duration 实时推进，
 * 不受本机与服务器时钟偏差影响）；未在构建时返回 null。
 */
export function executorProgress(row: ExecutorInfo): BuildProgress | null {
  return buildProgress({
    building: row.building,
    duration: row.duration,
    estimatedDuration: row.estimatedDuration,
  });
}

/** 构建号标签：等待启动时没有构建号，返回空串由调用方显示「—」。 */
export function executorJobLabel(row: ExecutorInfo): string {
  return row.buildNumber > 0 ? `#${row.buildNumber}` : "";
}
