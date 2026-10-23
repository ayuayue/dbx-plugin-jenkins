import { describe, expect, it } from "vitest";
import type { ExecutorInfo } from "../types";
import {
  executorJobLabel,
  executorNodeLabel,
  executorProgress,
  executorState,
  executorSummary,
} from "./executors";

function executor(overrides: Partial<ExecutorInfo> = {}): ExecutorInfo {
  return {
    node: "built-in",
    nodeOffline: false,
    index: 1,
    idle: true,
    jobName: "",
    buildNumber: 0,
    building: false,
    timestamp: 0,
    estimatedDuration: 0,
    duration: 0,
    result: "",
    pending: false,
    ...overrides,
  };
}

describe("executorNodeLabel", () => {
  it("控制器节点 displayName 为空时归一成内置节点文案", () => {
    expect(executorNodeLabel("", "内置节点")).toBe("内置节点");
    expect(executorNodeLabel("   ", "内置节点")).toBe("内置节点");
  });

  it("有名字的节点去掉首尾空白", () => {
    expect(executorNodeLabel(" agent-1 ", "内置节点")).toBe("agent-1");
  });
});

describe("executorState", () => {
  it("空闲执行器", () => {
    expect(executorState(executor({ idle: true }))).toBe("idle");
  });

  it("占位但构建尚未启动（Queue$Executable）", () => {
    expect(executorState(executor({ idle: false, pending: true }))).toBe("pending");
  });

  it("正在构建", () => {
    expect(
      executorState(executor({ idle: false, building: true, buildNumber: 12, jobName: "demo" })),
    ).toBe("building");
  });

  it("构建已结束但执行器还没释放", () => {
    expect(
      executorState(executor({ idle: false, building: false, buildNumber: 12, jobName: "demo" })),
    ).toBe("settling");
  });
});

describe("executorSummary", () => {
  it("空闲执行器计入总数，只把占用的算 busy", () => {
    const rows = [
      executor({ index: 1, idle: false, building: true, buildNumber: 3, jobName: "demo" }),
      executor({ index: 2, idle: true }),
      executor({ index: 3, idle: true }),
    ];
    expect(executorSummary(rows)).toEqual({ busy: 1, total: 3 });
  });

  it("等待启动的占位也算 busy", () => {
    expect(executorSummary([executor({ idle: false, pending: true })])).toEqual({
      busy: 1,
      total: 1,
    });
  });

  it("没有执行器时归零，不出现 NaN", () => {
    expect(executorSummary([])).toEqual({ busy: 0, total: 0 });
  });
});

describe("executorProgress", () => {
  it("构建中按服务端 duration / estimatedDuration 推进", () => {
    const progress = executorProgress(
      executor({ idle: false, building: true, duration: 4500, estimatedDuration: 10000 }),
    );
    expect(progress?.percent).toBe(45);
    expect(progress?.indeterminate).toBe(false);
  });

  it("超过估算时长也最多显示 99%", () => {
    const progress = executorProgress(
      executor({ idle: false, building: true, duration: 30000, estimatedDuration: 10000 }),
    );
    expect(progress?.percent).toBe(99);
  });

  it("没有估算耗时退化为不定态", () => {
    expect(
      executorProgress(executor({ idle: false, building: true, duration: 1000 })),
    ).toEqual({ percent: 0, indeterminate: true });
  });

  it("空闲或等待启动时没有进度", () => {
    expect(executorProgress(executor({ idle: true }))).toBeNull();
    expect(executorProgress(executor({ idle: false, pending: true }))).toBeNull();
  });
});

describe("executorJobLabel", () => {
  it("有构建号显示 #N", () => {
    expect(executorJobLabel(executor({ buildNumber: 12 }))).toBe("#12");
  });

  it("等待启动没有构建号", () => {
    expect(executorJobLabel(executor({ buildNumber: 0, pending: true }))).toBe("");
  });
});
