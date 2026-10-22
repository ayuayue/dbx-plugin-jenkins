import type { BuildSummary, StageMatrix } from "../types";

/** 构建列表轮询间隔：有构建中时每 3s 刷新一次。 */
export const AUTO_REFRESH_INTERVAL_MS = 3000;

/** 阶段视图轮询间隔：每次要拉最近 10 次构建的 wfapi，放宽到 5s。 */
export const STAGE_AUTO_REFRESH_INTERVAL_MS = 5000;

/** 触发构建后跟踪新构建出现的最长时间：排队过久就放弃跟踪，避免无休止轮询。 */
export const PENDING_BUILD_TIMEOUT_MS = 120_000;

const RUNNING_STAGE_STATUSES = new Set(["in_progress", "building"]);

/** 构建列表里是否存在构建中的构建。 */
export function hasRunningBuild(
  builds: BuildSummary[] | null | undefined,
): boolean {
  if (!builds?.length) return false;
  return builds.some((build) => build.building === true || build.status === "building");
}

/**
 * 阶段视图是否需要自动刷新：任一行构建中，或任一 stage 处于进行中。
 * 后端已把 wfapi 的 status 转成小写，这里再兜一层大小写。
 */
export function hasRunningStage(matrix: StageMatrix | null | undefined): boolean {
  if (!matrix?.builds?.length) return false;
  return matrix.builds.some((build) => {
    if (build.status === "building") return true;
    return (build.stages ?? []).some((stage) =>
      RUNNING_STAGE_STATUSES.has((stage.status ?? "").toLowerCase()),
    );
  });
}

/** 触发构建后的跟踪状态：等新构建出现。 */
export interface PendingBuild {
  /** 触发构建的任务全名。 */
  job: string;
  /** 触发前的最新构建号，用来判断是否真的出现了新构建。 */
  previousNumber: number;
  /** 跟踪截止时间戳（毫秒），到点仍未出现就放弃。 */
  deadline: number;
}

/** 记录一次「刚触发构建」，供构建列表主动刷新使用。 */
export function createPendingBuild(
  job: string,
  previousNumber: number,
  now: number = Date.now(),
): PendingBuild {
  return { job, previousNumber, deadline: now + PENDING_BUILD_TIMEOUT_MS };
}

/**
 * 是否已经能看到新构建：只要出现比触发前更大的构建号。
 * 不能用 `building` 兜底——触发时上一个构建可能还在跑，会被误判成新构建。
 */
export function hasNewBuild(
  builds: BuildSummary[] | null | undefined,
  previousNumber: number,
): boolean {
  if (!builds?.length) return false;
  return builds.some((build) => build.number > previousNumber);
}

export type PendingBuildStatus = "watching" | "settled" | "expired" | "stale";

/**
 * 判定待确认构建的跟踪状态：
 * - `stale`：任务已切换，跟踪作废；
 * - `settled`：新构建已出现，可以停止跟踪（构建中则由 autoRefresh 接手）；
 * - `expired`：超过截止时间仍未出现（如长时间排队），放弃跟踪；
 * - `watching`：继续等。
 */
export function evaluatePendingBuild(
  pending: PendingBuild,
  job: string,
  builds: BuildSummary[] | null | undefined,
  now: number = Date.now(),
): PendingBuildStatus {
  if (pending.job !== job) return "stale";
  if (hasNewBuild(builds, pending.previousNumber)) return "settled";
  if (now >= pending.deadline) return "expired";
  return "watching";
}

export interface AutoRefresherOptions {
  /** 两轮之间的间隔（毫秒）。 */
  intervalMs: number;
  /** 每轮执行的任务；抛出的异常不会打断后续轮询。 */
  task: () => Promise<void> | void;
  /** 每轮结束后判定是否继续；返回 false 则自动停止。 */
  shouldContinue?: () => boolean;
}

export interface AutoRefresher {
  start(): void;
  stop(): void;
  /** 重置计时（手动刷新后调用）；未启动时为空操作。 */
  restart(): void;
  readonly active: boolean;
}

/**
 * 串行轮询器：上一轮 await 完成后才排下一轮，天然不会并发叠加。
 * 定时器用全局 setTimeout，便于在 Node 测试环境注入 fake timers。
 */
export function createAutoRefresher(options: AutoRefresherOptions): AutoRefresher {
  const intervalMs = Math.max(250, options.intervalMs);
  let timer: ReturnType<typeof setTimeout> | undefined;
  let running = false;
  let generation = 0;

  function clearTimer(): void {
    if (timer !== undefined) {
      clearTimeout(timer);
      timer = undefined;
    }
  }

  function schedule(current: number): void {
    timer = setTimeout(() => {
      void tick(current);
    }, intervalMs);
  }

  async function tick(current: number): Promise<void> {
    timer = undefined;
    if (!running || current !== generation) return;

    try {
      await options.task();
    } catch {
      // 自动刷新失败不打断循环；是否提示由调用方决定。
    }

    if (!running || current !== generation) return;
    if (options.shouldContinue && !options.shouldContinue()) {
      running = false;
      return;
    }
    schedule(current);
  }

  function start(): void {
    if (running) return;
    running = true;
    generation += 1;
    schedule(generation);
  }

  function stop(): void {
    running = false;
    generation += 1;
    clearTimer();
  }

  function restart(): void {
    if (!running) return;
    generation += 1;
    clearTimer();
    schedule(generation);
  }

  return {
    start,
    stop,
    restart,
    get active() {
      return running;
    },
  };
}
