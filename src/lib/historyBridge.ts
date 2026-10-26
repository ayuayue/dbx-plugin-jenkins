/** 写进宿主 `history.state` 的命名空间键；宿主自己的 state 我们不碰也不读。 */
export const NAV_HISTORY_KEY = "dbxJenkinsNavIndex";

/** 只依赖用到的四个方法，方便在 node 环境注入假对象做单测。 */
interface HistoryLike {
  state: unknown;
  pushState(data: unknown, title: string, url?: string | null): void;
  replaceState(data: unknown, title: string, url?: string | null): void;
  go?(delta?: number): void;
}

export interface HistoryBridge {
  /** 能否安全操作宿主 history；一旦写入失败就永久降级。 */
  readonly available: boolean;
  /** 新增一条历史记录并写入游标。 */
  push(index: number): boolean;
  /** 只改当前条目（初始化用），不新增记录。 */
  replace(index: number): boolean;
  /** 让宿主栈移动一格；返回 false 表示不可用，调用方应直接改内部栈。 */
  go(delta: number): boolean;
  /** 读回游标；不是我们写的条目返回 null。 */
  index(): number | null;
}

function defaultHistory(): HistoryLike | null {
  if (typeof window === "undefined") return null;
  return (window.history as HistoryLike | undefined) ?? null;
}

/**
 * 宿主 history 的薄封装。
 *
 * 为什么必须 try/catch：`file://`（origin 为 null）和 `about:srcdoc` 下
 * `pushState` 会抛 SecurityError，插件单文件 HTML 在这两种方式下加载都出现过。
 * 一旦发现写不进去就永久降级——这时**绝不能**再调 `history.go`，
 * 因为浏览器栈里没有我们的条目，退一格会直接退出插件。
 */
export function createHistoryBridge(
  historyLike: HistoryLike | null = defaultHistory(),
): HistoryBridge {
  const target = historyLike;
  let usable = target !== null;

  const write = (index: number, kind: "push" | "replace"): boolean => {
    if (!usable || !target) return false;
    const state = { [NAV_HISTORY_KEY]: index };
    try {
      if (kind === "push") target.pushState(state, "");
      else target.replaceState(state, "");
      return true;
    } catch {
      // 永久降级：降级后每次导航再抛一次只会刷满控制台。
      usable = false;
      return false;
    }
  };

  return {
    get available() {
      return usable;
    },
    push: (index) => write(index, "push"),
    replace: (index) => write(index, "replace"),
    go(delta) {
      if (!usable || !target?.go) return false;
      target.go(delta);
      return true;
    },
    index() {
      const state = target?.state;
      if (!state || typeof state !== "object") return null;
      const value = (state as Record<string, unknown>)[NAV_HISTORY_KEY];
      return typeof value === "number" && Number.isFinite(value) ? value : null;
    },
  };
}
