import { describe, expect, it } from "vitest";
import type { ListViewFields } from "../types";
import {
  appendJobName,
  containsJob,
  isNamedView,
  needsJoinView,
  resolveJoinAction,
} from "./viewRefresh";

function fields(overrides: Partial<ListViewFields> = {}): ListViewFields {
  return {
    name: "Builds",
    description: "",
    jobNames: [],
    includeRegex: "",
    recurse: false,
    ...overrides,
  };
}

describe("isNamedView", () => {
  it("空名与 all（大小写不敏感）不算命名视图", () => {
    expect(isNamedView("")).toBe(false);
    expect(isNamedView("   ")).toBe(false);
    expect(isNamedView("all")).toBe(false);
    expect(isNamedView("ALL")).toBe(false);
    expect(isNamedView(null)).toBe(false);
    expect(isNamedView(undefined)).toBe(false);
  });

  it("自定义视图名算命名视图，首尾空白不影响", () => {
    expect(isNamedView("Builds")).toBe(true);
    expect(isNamedView("  运营视图  ")).toBe(true);
  });
});

describe("containsJob", () => {
  it("按 fullName 精确匹配，不做前缀或模糊匹配", () => {
    const jobs = [{ fullName: "team/demo" }, { fullName: "demo2" }];
    expect(containsJob(jobs, "team/demo")).toBe(true);
    expect(containsJob(jobs, "demo")).toBe(false);
    expect(containsJob([], "demo")).toBe(false);
    expect(containsJob(null, "demo")).toBe(false);
    expect(containsJob(undefined, "demo")).toBe(false);
  });

  it("空 fullName 不匹配任何任务", () => {
    expect(containsJob([{ fullName: "" }], "")).toBe(false);
  });
});

describe("appendJobName", () => {
  it("空集合追加为单元素", () => {
    expect(appendJobName([], "demo")).toEqual(["demo"]);
  });

  it("保持原顺序，末尾追加", () => {
    expect(appendJobName(["api", "web"], "demo")).toEqual(["api", "web", "demo"]);
  });

  it("已存在时不重复追加", () => {
    expect(appendJobName(["api", "demo"], "demo")).toEqual(["api", "demo"]);
  });

  it("jobNames 为 null 时按空集合处理（视图没有显式任务）", () => {
    expect(appendJobName(null, "demo")).toEqual(["demo"]);
    expect(appendJobName(undefined, "demo")).toEqual(["demo"]);
  });

  it("去掉空白项并对输入 trim", () => {
    expect(appendJobName(["api", "  "], "  demo  ")).toEqual(["api", "demo"]);
  });

  it("空任务名不写入", () => {
    expect(appendJobName(["api"], "   ")).toEqual(["api"]);
    expect(appendJobName(null, "")).toEqual([]);
  });
});

describe("needsJoinView", () => {
  it("根目录 + 命名视图 + 列表不含新任务时，需要写回视图", () => {
    expect(
      needsJoinView({
        folder: "",
        activeView: "Builds",
        fullName: "demo",
        jobs: [{ fullName: "api" }],
      }),
    ).toBe(true);
  });

  it("列表已包含新任务时不需要写（正则视图靠刷新即可）", () => {
    expect(
      needsJoinView({
        folder: "",
        activeView: "Builds",
        fullName: "demo",
        jobs: [{ fullName: "demo" }],
      }),
    ).toBe(false);
  });

  it("文件夹内新建任务不写视图", () => {
    expect(
      needsJoinView({
        folder: "team",
        activeView: "Builds",
        fullName: "team/demo",
        jobs: [],
      }),
    ).toBe(false);
  });

  it("全部视图或空视图名不写", () => {
    expect(
      needsJoinView({ folder: "", activeView: "all", fullName: "demo", jobs: [] }),
    ).toBe(false);
    expect(
      needsJoinView({ folder: "", activeView: "", fullName: "demo", jobs: [] }),
    ).toBe(false);
  });

  it("空任务名不触发写入", () => {
    expect(
      needsJoinView({ folder: "", activeView: "Builds", fullName: "  ", jobs: [] }),
    ).toBe(false);
  });
});

describe("resolveJoinAction", () => {
  it("不需要写回时直接跳过，不再读视图配置", () => {
    expect(resolveJoinAction(fields(), false)).toBe("skip");
  });

  it("需要写回且是 ListView 时执行追加", () => {
    expect(resolveJoinAction(fields(), true)).toBe("add");
    expect(resolveJoinAction(fields({ jobNames: null }), true)).toBe("add");
  });

  it("非 ListView（fields 为 null）无法写回，降级为提示", () => {
    expect(resolveJoinAction(null, true)).toBe("unsupported");
    expect(resolveJoinAction(undefined, true)).toBe("unsupported");
  });
});
