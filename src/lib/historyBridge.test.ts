import { describe, expect, it } from "vitest";
import { NAV_HISTORY_KEY, createHistoryBridge } from "./historyBridge";

interface FakeHistory {
  state: unknown;
  length: number;
  pushes: unknown[];
  replaces: unknown[];
  travelled: number[];
  pushState(data: unknown, title: string, url?: string | null): void;
  replaceState(data: unknown, title: string, url?: string | null): void;
  go(delta?: number): void;
}

function fakeHistory(failOn: "push" | "replace" | "both" | null = null): FakeHistory {
  const history: FakeHistory = {
    state: null,
    length: 1,
    pushes: [],
    replaces: [],
    travelled: [],
    pushState(data) {
      if (failOn === "push" || failOn === "both") {
        throw new Error("SecurityError: cannot pushState in a document with origin 'null'");
      }
      this.state = data;
      this.pushes.push(data);
      this.length += 1;
    },
    replaceState(data) {
      if (failOn === "replace" || failOn === "both") {
        throw new Error("SecurityError");
      }
      this.state = data;
      this.replaces.push(data);
    },
    go(delta) {
      this.travelled.push(delta ?? 0);
    },
  };
  return history;
}

describe("createHistoryBridge", () => {
  it("正常环境下 push 写入游标并返回 true", () => {
    const history = fakeHistory();
    const bridge = createHistoryBridge(history);
    expect(bridge.available).toBe(true);
    expect(bridge.push(3)).toBe(true);
    expect(history.state).toEqual({ [NAV_HISTORY_KEY]: 3 });
    expect(history.length).toBe(2);
  });

  it("replace 只改当前条目，不新增历史记录", () => {
    const history = fakeHistory();
    const bridge = createHistoryBridge(history);
    expect(bridge.replace(0)).toBe(true);
    expect(history.length).toBe(1);
    expect(bridge.index()).toBe(0);
  });

  it("pushState 抛错时降级：available 变 false，push 返回 false 且不抛出", () => {
    const history = fakeHistory("push");
    const bridge = createHistoryBridge(history);
    expect(() => bridge.push(1)).not.toThrow();
    expect(bridge.push(1)).toBe(false);
    expect(bridge.available).toBe(false);
    // 降级后不再尝试写，避免每次导航都抛一次。
    expect(bridge.push(2)).toBe(false);
    expect(history.pushes).toHaveLength(0);
  });

  it("replaceState 抛错同样降级", () => {
    const history = fakeHistory("replace");
    const bridge = createHistoryBridge(history);
    expect(bridge.replace(0)).toBe(false);
    expect(bridge.available).toBe(false);
  });

  it("没有 history 对象时直接不可用", () => {
    const bridge = createHistoryBridge(null);
    expect(bridge.available).toBe(false);
    expect(bridge.push(1)).toBe(false);
    expect(bridge.replace(0)).toBe(false);
    expect(bridge.go(-1)).toBe(false);
    expect(bridge.index()).toBeNull();
  });

  it("go 交给宿主栈移动，返回 true", () => {
    const history = fakeHistory();
    const bridge = createHistoryBridge(history);
    expect(bridge.go(-1)).toBe(true);
    expect(bridge.go(1)).toBe(true);
    expect(history.travelled).toEqual([-1, 1]);
  });

  it("降级后 go 返回 false，绝不委托宿主（file:// 下 back 会离开插件）", () => {
    const history = fakeHistory("push");
    const bridge = createHistoryBridge(history);
    bridge.push(1);
    expect(bridge.available).toBe(false);
    expect(bridge.go(-1)).toBe(false);
    expect(history.travelled).toEqual([]);
  });

  it("index 只认自家命名空间下的数字", () => {
    const history = fakeHistory();
    const bridge = createHistoryBridge(history);
    expect(bridge.index()).toBeNull();

    history.state = { [NAV_HISTORY_KEY]: 7 };
    expect(bridge.index()).toBe(7);

    // 宿主自己的 history state 不能被误读成游标。
    history.state = { someHostKey: 42 };
    expect(bridge.index()).toBeNull();

    history.state = { [NAV_HISTORY_KEY]: "7" };
    expect(bridge.index()).toBeNull();
  });

  it("push 之后 index 读回刚写入的游标", () => {
    const history = fakeHistory();
    const bridge = createHistoryBridge(history);
    bridge.push(5);
    expect(bridge.index()).toBe(5);
  });
});
