import { describe, expect, it } from "vitest";
import {
  ENTER_REFRESH_MIN_GAP_MS,
  resolveEnterActions,
  shouldRefreshOnEnter,
} from "./enterRefresh";

describe("resolveEnterActions", () => {
  it("概览页刷新概览数据 + 执行器列表，队列 / 节点页只刷自己那一份", () => {
    const state = { hasViews: true, hasSelection: true };
    expect(resolveEnterActions("overview", state)).toEqual(["overview", "executors"]);
    expect(resolveEnterActions("queue", state)).toEqual(["queue"]);
    expect(resolveEnterActions("nodes", state)).toEqual(["nodes"]);
  });

  it("任务页首次进入要先拉视图列表、再拉任务", () => {
    expect(resolveEnterActions("jobs", { hasViews: false, hasSelection: false })).toEqual([
      "views",
      "jobs",
    ]);
  });

  it("任务页已有视图时只需刷任务列表", () => {
    expect(resolveEnterActions("jobs", { hasViews: true, hasSelection: false })).toEqual([
      "jobs",
    ]);
  });

  it("任务页有选中任务时，构建列表也要一起刷新", () => {
    expect(resolveEnterActions("jobs", { hasViews: true, hasSelection: true })).toEqual([
      "jobs",
      "detail",
    ]);
  });

  it("三项都缺时顺序固定为 views → jobs → detail", () => {
    expect(resolveEnterActions("jobs", { hasViews: false, hasSelection: true })).toEqual([
      "views",
      "jobs",
      "detail",
    ]);
  });

  it("构建详情与阶段页不参与进入刷新：内容由打开动作自己加载", () => {
    const state = { hasViews: true, hasSelection: true };
    expect(resolveEnterActions("build", state)).toEqual([]);
    expect(resolveEnterActions("stages", state)).toEqual([]);
  });
});

describe("shouldRefreshOnEnter", () => {
  it("换页面总是刷新，不受上次刷新时间影响", () => {
    const now = 10_000;
    expect(shouldRefreshOnEnter("jobs", { view: "overview", at: now - 100 }, now)).toBe(true);
  });

  it("首次进入某页（还没有上次记录）总是刷新", () => {
    const now = 10_000;
    expect(shouldRefreshOnEnter("jobs", { view: null, at: now - 100 }, now)).toBe(true);
    expect(shouldRefreshOnEnter("jobs", { view: "jobs", at: null }, now)).toBe(true);
    expect(shouldRefreshOnEnter("jobs", { view: "jobs", at: 0 }, now)).toBe(true);
  });

  it("同一页在最小间隔内重复进入不重复发请求", () => {
    const now = 10_000;
    expect(shouldRefreshOnEnter("jobs", { view: "jobs", at: now - 200 }, now)).toBe(false);
    expect(
      shouldRefreshOnEnter(
        "jobs",
        { view: "jobs", at: now - ENTER_REFRESH_MIN_GAP_MS + 1 },
        now,
      ),
    ).toBe(false);
  });

  it("同一页刚好到达最小间隔就放行", () => {
    const now = 10_000;
    expect(
      shouldRefreshOnEnter("jobs", { view: "jobs", at: now - ENTER_REFRESH_MIN_GAP_MS }, now),
    ).toBe(true);
    expect(
      shouldRefreshOnEnter(
        "jobs",
        { view: "jobs", at: now - ENTER_REFRESH_MIN_GAP_MS - 1 },
        now,
      ),
    ).toBe(true);
  });

  it("支持自定义间隔；间隔为 0 表示不节流", () => {
    const now = 10_000;
    const gate = { view: "jobs" as const, at: now - 500 };
    expect(shouldRefreshOnEnter("jobs", gate, now, 0)).toBe(true);
    expect(shouldRefreshOnEnter("jobs", gate, now, 800)).toBe(false);
  });

  it("时钟回拨（上次时间在未来）不刷", () => {
    expect(shouldRefreshOnEnter("jobs", { view: "jobs", at: 20_000 }, 10_000)).toBe(false);
  });
});
