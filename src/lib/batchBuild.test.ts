import { describe, expect, it } from "vitest";
import type {
  BatchBuildResult,
  BatchTargetInfo,
  JobSummary,
  ParameterDefinition,
} from "../types";
import {
  BATCH_CHUNK_SIZE,
  MAX_BATCH_JOBS,
  abortedChunk,
  batchProgress,
  batchReasonKey,
  batchTargets,
  buildParamValues,
  chunkJobs,
  collectSharedParameters,
  countOutsideCurrent,
  failedChunk,
  mergeBatchResults,
  rememberTargets,
  selectRange,
  selectVisible,
  shouldAbortChunks,
  summarizeBatch,
  toggleJobSelection,
} from "./batchBuild";

function job(name: string, overrides: Partial<JobSummary> = {}): JobSummary {
  return {
    name,
    fullName: name,
    url: `http://ci/job/${name}/`,
    color: "blue",
    isFolder: false,
    status: "success",
    building: false,
    lastBuild: null,
    ...overrides,
  };
}

function result(jobName: string, overrides: Partial<BatchBuildResult> = {}): BatchBuildResult {
  return {
    job: jobName,
    success: true,
    skipped: false,
    queueId: 1,
    reason: "",
    error: "",
    ...overrides,
  };
}

describe("toggleJobSelection", () => {
  it("追加新任务并按点击顺序保留", () => {
    expect(toggleJobSelection([], "a")).toEqual(["a"]);
    expect(toggleJobSelection(["a"], "b")).toEqual(["a", "b"]);
  });

  it("再次点击取消勾选", () => {
    expect(toggleJobSelection(["a", "b"], "a")).toEqual(["b"]);
    expect(toggleJobSelection(["a"], "a")).toEqual([]);
  });

  it("达到上限后不再追加，但仍可取消", () => {
    expect(toggleJobSelection(["a", "b"], "c", 2)).toEqual(["a", "b"]);
    expect(toggleJobSelection(["a", "b"], "a", 2)).toEqual(["b"]);
  });

  it("默认上限是 50", () => {
    expect(MAX_BATCH_JOBS).toBe(50);
  });
});

describe("selectVisible", () => {
  it("只选当前列表里的任务，跳过文件夹", () => {
    const jobs = [job("team", { isFolder: true }), job("a"), job("b")];
    expect(selectVisible([], jobs)).toEqual(["a", "b"]);
  });

  it("与已有选择合并并去重", () => {
    const jobs = [job("b"), job("c")];
    expect(selectVisible(["a", "b"], jobs)).toEqual(["a", "b", "c"]);
  });

  it("受上限约束", () => {
    const jobs = [job("b"), job("c"), job("d")];
    expect(selectVisible(["a"], jobs, 2)).toEqual(["a", "b"]);
  });
});

describe("rememberTargets", () => {
  it("合并当前列表并保留跨视图的旧项", () => {
    const first = rememberTargets({}, [job("a"), job("b")]);
    const second = rememberTargets(first, [
      job("b", { building: true, status: "building" }),
      job("c"),
    ]);
    expect(Object.keys(second).sort()).toEqual(["a", "b", "c"]);
    expect(second.b.building).toBe(true);
    expect(second.a.name).toBe("a");
  });

  it("文件夹不进入缓存", () => {
    const map = rememberTargets({}, [job("team", { isFolder: true })]);
    expect(map).toEqual({});
  });
});

describe("batchTargets", () => {
  function targetMap() {
    return rememberTargets({}, [job("a"), job("b")]);
  }

  it("按选择顺序生成清单", () => {
    expect(batchTargets(["b", "a"], targetMap())).toEqual([
      { fullName: "b", name: "b", status: "success", building: false },
      { fullName: "a", name: "a", status: "success", building: false },
    ]);
  });

  it("忽略已不存在的选中项", () => {
    expect(batchTargets(["a", "gone"], targetMap())).toHaveLength(1);
  });
});

describe("countOutsideCurrent", () => {
  it("统计不在当前列表里的已选任务", () => {
    const jobs = [job("a"), job("b")];
    expect(countOutsideCurrent(["a", "b"], jobs)).toBe(0);
    expect(countOutsideCurrent(["a", "x", "y"], jobs)).toBe(2);
  });
});

describe("buildParamValues", () => {
  it("按类型初始化默认值", () => {
    const values = buildParamValues([
      { name: "BRANCH", type: "StringParameterDefinition", description: "", defaultValue: "main" },
      { name: "DRY_RUN", type: "BooleanParameterDefinition", description: "", defaultValue: "false" },
      { name: "EMPTY", type: "StringParameterDefinition", description: "", defaultValue: "" },
    ]);
    expect(values).toEqual({ BRANCH: "main", DRY_RUN: false, EMPTY: "" });
  });
});

describe("collectSharedParameters", () => {
  function target(name: string, parameters: ParameterDefinition[] | null): BatchTargetInfo {
    return {
      name,
      fullName: name,
      status: "success",
      building: false,
      buildable: true,
      inQueue: false,
      parameters,
    };
  }

  it("只保留 Boolean 开关，其余参数不参与共享", () => {
    const shared = collectSharedParameters([
      target("a", [
        {
          name: "MODULE_NAME",
          type: "ChoiceParameterDefinition",
          description: "",
          defaultValue: "modules-a",
          choices: ["modules-a"],
        },
        { name: "BRANCH_NAME", type: "StringParameterDefinition", description: "", defaultValue: "test" },
        { name: "SKIP_BUILD", type: "BooleanParameterDefinition", description: "", defaultValue: "false" },
      ]),
    ]);
    expect(shared.parameters.map((item) => item.name)).toEqual(["SKIP_BUILD"]);
  });

  it("多任务同名开关去重，保留首次出现的默认值", () => {
    const shared = collectSharedParameters([
      target("a", [
        { name: "SKIP_BUILD", type: "BooleanParameterDefinition", description: "", defaultValue: "false" },
      ]),
      target("b", [
        { name: "SKIP_BUILD", type: "BooleanParameterDefinition", description: "", defaultValue: "true" },
        {
          name: "SKIP_CODE_CHECKOUT",
          type: "BooleanParameterDefinition",
          description: "",
          defaultValue: "false",
        },
      ]),
    ]);
    expect(shared.parameters.map((item) => item.name)).toEqual([
      "SKIP_BUILD",
      "SKIP_CODE_CHECKOUT",
    ]);
    expect(shared.parameters[0].defaultValue).toBe("false");
  });

  it("没有任何开关参数时返回空（参数区隐藏）", () => {
    const shared = collectSharedParameters([
      target("a", [
        { name: "BRANCH_NAME", type: "StringParameterDefinition", description: "", defaultValue: "test" },
      ]),
      target("b", null),
    ]);
    expect(shared.parameters).toEqual([]);
  });
});

describe("summarizeBatch", () => {
  it("分别统计已触发 / 跳过 / 失败", () => {
    const summary = summarizeBatch([
      result("a"),
      result("b", { skipped: true, reason: "in_queue" }),
      result("c", { success: false, reason: "forbidden", error: "forbidden: nope" }),
    ]);
    expect(summary.queued).toBe(1);
    expect(summary.skipped).toBe(1);
    expect(summary.failed).toBe(1);
    expect(summary.failures.map((item) => item.job)).toEqual(["c"]);
  });

  it("空结果全为 0", () => {
    expect(summarizeBatch([])).toEqual({ queued: 0, skipped: 0, failed: 0, failures: [] });
  });
});

describe("batchReasonKey", () => {
  it("已知原因映射到对应文案", () => {
    expect(batchReasonKey("not_buildable")).toBe("batchReasonNotBuildable");
    expect(batchReasonKey("in_queue")).toBe("batchReasonInQueue");
    expect(batchReasonKey("building")).toBe("batchReasonBuilding");
  });

  it("失败类原因单独给出文案", () => {
    expect(batchReasonKey("forbidden")).toBe("batchReasonForbidden");
    expect(batchReasonKey("not_found")).toBe("batchReasonNotFound");
    expect(batchReasonKey("read_only")).toBe("batchReasonReadOnly");
    expect(batchReasonKey("rejected")).toBe("batchReasonRejected");
    expect(batchReasonKey("timeout")).toBe("batchReasonTimeout");
    expect(batchReasonKey("aborted")).toBe("batchReasonAborted");
  });

  it("未知原因回落到通用失败文案", () => {
    expect(batchReasonKey("weird")).toBe("batchBuildError");
    expect(batchReasonKey("")).toBe("batchBuildError");
  });
});

describe("chunkJobs", () => {
  it("按 10 个一批切开，最后一批保留余数", () => {
    const names = Array.from({ length: 25 }, (_, index) => `job-${index}`);
    const chunks = chunkJobs(names);
    expect(chunks.map((chunk) => chunk.length)).toEqual([10, 10, 5]);
    expect(chunks[0][0]).toBe("job-0");
    expect(chunks[2][4]).toBe("job-24");
  });

  it("空列表返回空批次", () => {
    expect(chunkJobs([])).toEqual([]);
  });

  it("非法批大小回落到默认 10", () => {
    const names = Array.from({ length: 12 }, (_, index) => `job-${index}`);
    expect(chunkJobs(names, 0).map((chunk) => chunk.length)).toEqual([10, 2]);
    expect(chunkJobs(names, -3).map((chunk) => chunk.length)).toEqual([10, 2]);
  });

  it("默认批大小为 10", () => {
    expect(BATCH_CHUNK_SIZE).toBe(10);
  });
});

describe("mergeBatchResults", () => {
  it("按批次顺序拼接", () => {
    const merged = mergeBatchResults(
      [result("a")],
      [result("b"), result("c", { skipped: true, reason: "in_queue" })],
    );
    expect(merged.map((item) => item.job)).toEqual(["a", "b", "c"]);
  });

  it("没有批次时返回空数组", () => {
    expect(mergeBatchResults()).toEqual([]);
  });
});

describe("failedChunk", () => {
  it("整批标记为请求失败并保留错误原文", () => {
    const chunk = failedChunk(["a", "b"], "request_failed: timeout");
    expect(chunk).toHaveLength(2);
    expect(chunk[0]).toMatchObject({
      job: "a",
      success: false,
      skipped: false,
      queueId: 0,
      reason: "request_failed",
      error: "request_failed: timeout",
    });
    expect(chunk[1].job).toBe("b");
  });
});

describe("abortedChunk", () => {
  it("未执行的任务标记为跳过", () => {
    expect(abortedChunk(["a"])).toEqual([
      {
        job: "a",
        success: false,
        skipped: true,
        queueId: 0,
        reason: "aborted",
        error: "",
      },
    ]);
  });
});

describe("batchProgress", () => {
  it("按完成比例给出整数百分比", () => {
    expect(batchProgress(0, 30)).toBe(0);
    expect(batchProgress(12, 30)).toBe(40);
    expect(batchProgress(30, 30)).toBe(100);
    expect(batchProgress(1, 3)).toBe(33);
  });

  it("边界安全：不除零、不超 100", () => {
    expect(batchProgress(0, 0)).toBe(0);
    expect(batchProgress(5, 3)).toBe(100);
  });
});

describe("shouldAbortChunks", () => {
  it("连续 2 批失败才中止剩余", () => {
    expect(shouldAbortChunks(0)).toBe(false);
    expect(shouldAbortChunks(1)).toBe(false);
    expect(shouldAbortChunks(2)).toBe(true);
  });
});

describe("selectRange", () => {
  const visible = ["a", "b", "c", "d"];

  it("正向区间加选", () => {
    expect(selectRange([], visible, "a", "c")).toEqual(["a", "b", "c"]);
  });

  it("反向区间同样加选", () => {
    expect(selectRange([], visible, "c", "a")).toEqual(["a", "b", "c"]);
  });

  it("已选项保留在前面且不重复", () => {
    expect(selectRange(["d"], visible, "a", "b")).toEqual(["d", "a", "b"]);
  });

  it("anchor 失效时退化成单点勾选", () => {
    expect(selectRange([], visible, "gone", "d")).toEqual(["d"]);
  });

  it("受上限约束", () => {
    expect(selectRange([], visible, "a", "d", 2)).toEqual(["a", "b"]);
  });
});
