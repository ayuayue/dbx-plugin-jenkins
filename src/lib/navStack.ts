import type { AppView } from "./enterRefresh";

/**
 * 一条可回放的导航记录。
 *
 * 五个字段足以还原任何一屏：页面、视图 tab、文件夹、选中任务、构建号。
 * 刻意不存整份 JobDetail / BuildDetail —— 那些是数据，回放时重新拉，
 * 否则历史和真实状态会各自漂移。
 */
export interface NavEntry {
  view: AppView;
  /** 当前 Jenkins 视图名；空串表示「全部任务」。 */
  activeView: string;
  /** 任务列表所在文件夹路径。 */
  folder: string;
  /** 选中任务的 fullName；空串表示没有选中。 */
  job: string;
  /** 仅 view === "build" 时有值。 */
  build: number | null;
}

/** push 的结果：added 为 false 表示这次没有真正新增记录（重复条目）。 */
export interface NavPushResult {
  index: number;
  added: boolean;
}

/** 历史深度上限：够退回十几步，又不至于把整条浏览路径无限堆在内存里。 */
export const NAV_STACK_LIMIT = 50;

export interface NavStack {
  size(): number;
  index(): number;
  current(): NavEntry;
  canBack(): boolean;
  canForward(): boolean;
  /** 记录一条新导航；与当前条目相同时不新增。 */
  push(entry: NavEntry): NavPushResult;
  /** 后退一步，返回新游标；已在最早一条时返回 null。 */
  back(): number | null;
  /** 前进一步，返回新游标；已在本条时返回 null。 */
  forward(): number | null;
  /** 按游标恢复（popstate 用），越界夹紧，返回该条目。 */
  restore(index: number): NavEntry;
  /** 重置整栈，只保留一条记录。 */
  reset(entry: NavEntry): void;
}

/** 五个字段全等才算同一屏：少比一个就会出现「退不动」或者多出重复记录。 */
function sameEntry(a: NavEntry, b: NavEntry): boolean {
  return (
    a.view === b.view &&
    a.activeView === b.activeView &&
    a.folder === b.folder &&
    a.job === b.job &&
    a.build === b.build
  );
}

export function createNavStack(
  initial: NavEntry,
  limit: number = NAV_STACK_LIMIT,
): NavStack {
  const max = Math.max(1, Math.floor(limit) || 1);
  const entries: NavEntry[] = [initial];
  let cursor = 0;

  return {
    size: () => entries.length,
    index: () => cursor,
    current: () => entries[cursor],
    canBack: () => cursor > 0,
    canForward: () => cursor < entries.length - 1,

    push(entry) {
      // 连点当前任务、重复点同一个视图 tab 都不该产生记录：
      // 否则用户要多按一次后退才能回去。
      if (sameEntry(entries[cursor], entry)) return { index: cursor, added: false };

      // 在中间位置导航会截断原来的前进分支——这是浏览器历史的标准语义。
      entries.splice(cursor + 1);
      entries.push(entry);
      if (entries.length > max) entries.shift();
      cursor = entries.length - 1;
      return { index: cursor, added: true };
    },

    back() {
      if (cursor === 0) return null;
      cursor -= 1;
      return cursor;
    },

    forward() {
      if (cursor >= entries.length - 1) return null;
      cursor += 1;
      return cursor;
    },

    restore(index) {
      // 浏览器 history 里可能留着上一份文档的游标，甚至宿主自己塞的脏值，一律夹紧。
      const next = Number.isFinite(index) ? Math.round(index) : 0;
      cursor = Math.min(Math.max(next, 0), entries.length - 1);
      return entries[cursor];
    },

    reset(entry) {
      entries.length = 0;
      entries.push(entry);
      cursor = 0;
    },
  };
}
