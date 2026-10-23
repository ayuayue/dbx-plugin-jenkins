/** 应用内的页面标识（构建详情与阶段视图依赖选中任务，不参与进入刷新）。 */
export type AppView = "overview" | "jobs" | "build" | "stages" | "queue" | "nodes";

/** 进入页面时需要补刷的数据源。 */
export type EnterRefreshAction =
  | "views"
  | "jobs"
  | "detail"
  | "overview"
  | "executors"
  | "queue"
  | "nodes";

export interface EnterRefreshState {
  /** 视图列表是否已加载过。 */
  hasViews: boolean;
  /** 是否已选中任务（决定要不要连带刷构建列表）。 */
  hasSelection: boolean;
}

/**
 * 切换页面时需要刷新哪些数据。
 *
 * 返回空数组表示这一页不用额外请求：构建详情与阶段视图的内容由打开动作
 * 自己加载，返回列表的路径另有刷新。
 *
 * 概览页顺序固定为 overview → executors：概览拿的是聚合数字，执行器列表
 * 是它的明细，先出数字再出明细，视觉上不会跳。
 * 任务页的顺序固定为 views → jobs → detail：先拿到视图（视图决定任务集合）、
 * 再拉任务、最后刷选中任务的构建列表。
 */
export function resolveEnterActions(
  view: AppView,
  state: EnterRefreshState,
): EnterRefreshAction[] {
  if (view === "overview") return ["overview", "executors"];
  if (view === "queue") return ["queue"];
  if (view === "nodes") return ["nodes"];
  if (view !== "jobs") return [];

  const actions: EnterRefreshAction[] = [];
  if (!state.hasViews) actions.push("views");
  actions.push("jobs");
  if (state.hasSelection) actions.push("detail");
  return actions;
}

/**
 * 进入刷新的最小间隔：同一页快速连点时只发一次请求。
 * 自动轮询刚刷过也算「数据是新的」，因此共用同一个时间戳。
 */
export const ENTER_REFRESH_MIN_GAP_MS = 2000;

export interface EnterRefreshGate {
  /** 上次执行进入刷新的页面。 */
  view: AppView | null;
  /** 上次成功刷新的时间戳（与自动轮询共用）；0 / null 表示从未刷新。 */
  at: number | null | undefined;
}

/**
 * 是否该为这次「进入页面」发请求。
 *
 * 只有**同一页**的重复进入才节流：换页必刷，否则首次点开任务页会被
 * 刚刚加载过概览的时间戳挡掉，列表直接空白。时钟回拨（上次时间在未来）
 * 按不可信处理，不重复请求。
 */
export function shouldRefreshOnEnter(
  view: AppView,
  gate: EnterRefreshGate,
  now: number,
  minGap: number = ENTER_REFRESH_MIN_GAP_MS,
): boolean {
  if (gate.view !== view) return true;
  if (!gate.at) return true;
  if (now < gate.at) return false;
  return now - gate.at >= Math.max(0, minGap);
}
