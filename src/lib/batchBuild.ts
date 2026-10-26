import type { MessageKey } from "../i18n";
import type {
  BatchBuildResult,
  BatchTargetInfo,
  JenkinsStatus,
  JobSummary,
  ParameterDefinition,
} from "../types";

/** 单次批量构建的任务上限：够发版用，又能挡住「全选几百个」的误触。 */
export const MAX_BATCH_JOBS = 50;

/** 单批任务数：10 个约 1–2s 一跳，进度够细，请求也不至于太碎。 */
export const BATCH_CHUNK_SIZE = 10;

/** 批量构建确认弹窗里的一行。 */
export interface BatchTarget {
  fullName: string;
  name: string;
  status: JenkinsStatus;
  building: boolean;
}

/** 跨视图的已选任务缓存：切到别的视图后仍能展示名字与状态。 */
export type BatchTargetMap = Record<string, BatchTarget>;

export interface BatchSummary {
  queued: number;
  skipped: number;
  failed: number;
  /** 失败明细（不含被跳过的）。 */
  failures: BatchBuildResult[];
}

const REASON_KEYS: Record<string, MessageKey> = {
  not_buildable: "batchReasonNotBuildable",
  in_queue: "batchReasonInQueue",
  building: "batchReasonBuilding",
  forbidden: "batchReasonForbidden",
  not_found: "batchReasonNotFound",
  read_only: "batchReasonReadOnly",
  rejected: "batchReasonRejected",
  timeout: "batchReasonTimeout",
  aborted: "batchReasonAborted",
};

/** 跳过/失败原因 → 文案 key；未知原因（如 internal）回落到通用失败文案。 */
export function batchReasonKey(reason: string): MessageKey {
  return REASON_KEYS[reason] ?? "batchBuildError";
}

/** 按固定大小切批；非法大小回落到默认值。 */
export function chunkJobs<T>(list: T[], size: number = BATCH_CHUNK_SIZE): T[][] {
  const width = size > 0 ? size : BATCH_CHUNK_SIZE;
  const chunks: T[][] = [];
  for (let index = 0; index < list.length; index += width) {
    chunks.push(list.slice(index, index + width));
  }
  return chunks;
}

/** 按批次顺序拼接结果，保持与任务顺序一致。 */
export function mergeBatchResults(...groups: BatchBuildResult[][]): BatchBuildResult[] {
  const merged: BatchBuildResult[] = [];
  for (const group of groups) {
    merged.push(...group);
  }
  return merged;
}

/** 整批请求失败（超时/断网）时，把这一批全部标成失败，保住已触发的结果。 */
export function failedChunk(names: string[], message: string): BatchBuildResult[] {
  return names.map((job) => ({
    job,
    success: false,
    skipped: false,
    queueId: 0,
    reason: "request_failed",
    error: message,
  }));
}

/** 连续失败达到阈值后，剩余批次不再发送，标成「未执行」。 */
export function abortedChunk(names: string[]): BatchBuildResult[] {
  return names.map((job) => ({
    job,
    success: false,
    skipped: true,
    queueId: 0,
    reason: "aborted",
    error: "",
  }));
}

/** 完成百分比：整数、0–100，`total=0` 不除零。 */
export function batchProgress(done: number, total: number): number {
  if (total <= 0) return 0;
  const percent = Math.round((done / total) * 100);
  return Math.min(100, Math.max(0, percent));
}

/** 连续失败达到 max（默认 2）批就中止剩余。 */
export function shouldAbortChunks(consecutiveFailures: number, max = 2): boolean {
  return consecutiveFailures >= max;
}

/** 勾选/取消一个任务；达到上限后不再追加（仍允许取消）。 */
export function toggleJobSelection(
  selected: string[],
  fullName: string,
  max: number = MAX_BATCH_JOBS,
): string[] {
  if (selected.includes(fullName)) {
    return selected.filter((name) => name !== fullName);
  }
  if (selected.length >= max) {
    return selected;
  }
  return [...selected, fullName];
}

/** 全选当前列表里的任务（文件夹不参与），受上限约束。 */
export function selectVisible(
  selected: string[],
  jobs: JobSummary[],
  max: number = MAX_BATCH_JOBS,
): string[] {
  const chosen = [...selected];
  for (const job of jobs) {
    if (job.isFolder || chosen.includes(job.fullName)) continue;
    if (chosen.length >= max) break;
    chosen.push(job.fullName);
  }
  return chosen;
}

/** 刷新/重载后把当前列表并入跨视图缓存：名字与状态始终取最新一次看到的值。 */
export function rememberTargets(
  map: BatchTargetMap,
  jobs: JobSummary[],
): BatchTargetMap {
  const next: BatchTargetMap = { ...map };
  for (const job of jobs) {
    if (job.isFolder) continue;
    next[job.fullName] = {
      fullName: job.fullName,
      name: job.name,
      status: job.status,
      building: job.building,
    };
  }
  return next;
}

/** 选中项 → 弹窗清单：按勾选顺序输出，忽略已不存在的任务。 */
export function batchTargets(
  selected: string[],
  map: BatchTargetMap,
): BatchTarget[] {
  const targets: BatchTarget[] = [];
  for (const fullName of selected) {
    const target = map[fullName];
    if (target) targets.push(target);
  }
  return targets;
}

/** 有多少已选任务不在当前列表里（跨视图选择时提示用）。 */
export function countOutsideCurrent(selected: string[], jobs: JobSummary[]): number {
  const visible = new Set(
    jobs.filter((job) => !job.isFolder).map((job) => job.fullName),
  );
  return selected.filter((name) => !visible.has(name)).length;
}

/** 汇总批量结果：已触发 / 跳过 / 失败 + 失败明细。 */
export function summarizeBatch(results: BatchBuildResult[]): BatchSummary {
  const summary: BatchSummary = { queued: 0, skipped: 0, failed: 0, failures: [] };
  for (const item of results) {
    if (item.skipped) {
      summary.skipped += 1;
      continue;
    }
    if (item.success) {
      summary.queued += 1;
      continue;
    }
    summary.failed += 1;
    summary.failures.push(item);
  }
  return summary;
}

/**
 * Shift 点选：把 anchor 到 target 的可见任务按区间加选。
 * anchor 已不在可见列表里时退化成单点勾选，避免误选一片。
 */
export function selectRange(
  selected: string[],
  visible: string[],
  anchor: string,
  target: string,
  max: number = MAX_BATCH_JOBS,
): string[] {
  const from = visible.indexOf(anchor);
  const to = visible.indexOf(target);
  if (from < 0 || to < 0) {
    return toggleJobSelection(selected, target, max);
  }

  const next = [...selected];
  const start = Math.min(from, to);
  const end = Math.max(from, to);
  for (const name of visible.slice(start, end + 1)) {
    if (next.includes(name)) continue;
    if (next.length >= max) break;
    next.push(name);
  }
  return next;
}

/** 参数默认值 → 表单初始值（boolean 用布尔，其余用字符串）。 */
export function buildParamValues(
  parameters: ParameterDefinition[],
): Record<string, unknown> {
  const values: Record<string, unknown> = {};
  for (const parameter of parameters) {
    values[parameter.name] =
      parameter.type === "BooleanParameterDefinition"
        ? parameter.defaultValue === "true"
        : (parameter.defaultValue ?? "");
  }
  return values;
}

export interface SharedParameters {
  /** 只含开关（Boolean）参数，按首次出现顺序去重。 */
  parameters: ParameterDefinition[];
}

/** 批量只共享开关类参数：String/Choice 多为任务级差异化配置，共用一个值会出错。 */
const SHARED_PARAMETER_TYPE = "BooleanParameterDefinition";

/**
 * 把所选任务的**开关参数**按名称做并集，供批量构建一次性填写。
 * 同名参数保留首次出现的定义；其余类型（String/Choice/File…）不参与共享，
 * 构建时各任务使用自己的默认值。
 */
export function collectSharedParameters(targets: BatchTargetInfo[]): SharedParameters {
  const byName = new Map<string, ParameterDefinition>();

  for (const target of targets) {
    for (const parameter of target.parameters ?? []) {
      if (parameter.type !== SHARED_PARAMETER_TYPE) continue;
      if (!byName.has(parameter.name)) {
        byName.set(parameter.name, parameter);
      }
    }
  }

  return { parameters: [...byName.values()] };
}
