import type { BuildSummary, StageInfo } from "../types";

export interface BuildProgress {
  /** 构建中已推进的百分比，0-99；不确定态固定 0。 */
  percent: number;
  /** 缺少 estimatedDuration 时无法算百分比，用条纹动画表示「进行中」。 */
  indeterminate: boolean;
}

/** 运行中的构建最多显示到 99%，避免没跑完就满格。 */
const MAX_RUNNING_PERCENT = 99;

/** 已完成（或跳过）的阶段算一格。 */
const STAGE_DONE = new Set(["success", "skipped"]);
/** 正在执行的阶段算半格，避免长阶段一直卡在上一格。 */
const STAGE_RUNNING = new Set(["in_progress", "paused"]);

export type StageStatus = Pick<StageInfo, "status">;

/**
 * 按阶段步进算进度：`(已完成 + 进行中×0.5) / 阶段总数`。
 *
 * Pipeline 的阶段数在运行中会变（动态 stage / parallel），所以不保证单调递增；
 * 出现明显回退再补「只增不减」的记忆逻辑。
 */
export function stageProgress(stages: StageStatus[]): BuildProgress | null {
  if (!stages.length) return null;

  let done = 0;
  let running = 0;
  for (const stage of stages) {
    const status = (stage.status ?? "").toLowerCase();
    if (STAGE_DONE.has(status)) {
      done++;
    } else if (STAGE_RUNNING.has(status)) {
      running++;
    }
  }

  const ratio = (done + running * 0.5) / stages.length;
  return {
    percent: Math.min(MAX_RUNNING_PERCENT, Math.max(0, Math.floor(ratio * 100))),
    indeterminate: false,
  };
}

/**
 * 计算构建进度；已完成的构建返回 null（列表显示「—」）。
 *
 * 有阶段数据时按阶段步进（更贴近 Pipeline 真实进度），否则回退到时长估算：
 * 用 Jenkins 返回的 duration 而不是本地时间减 timestamp —— duration 由服务端
 * 在构建期间实时推进，不受本机与服务器时钟偏差影响。
 */
export function buildProgress(
  build: Pick<BuildSummary, "building" | "duration" | "estimatedDuration">,
  stages?: StageStatus[] | null,
): BuildProgress | null {
  if (!build.building) return null;

  if (stages?.length) {
    const byStage = stageProgress(stages);
    if (byStage) return byStage;
  }

  const estimate = build.estimatedDuration;
  if (!Number.isFinite(estimate) || estimate <= 0) {
    return { percent: 0, indeterminate: true };
  }

  const elapsed = Number.isFinite(build.duration) ? Math.max(0, build.duration) : 0;
  const percent = Math.min(MAX_RUNNING_PERCENT, Math.floor((elapsed / estimate) * 100));
  return { percent: Math.max(0, percent), indeterminate: false };
}
