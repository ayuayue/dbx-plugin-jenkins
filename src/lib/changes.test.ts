import { describe, expect, it } from "vitest";
import {
  changeBody,
  changeSummary,
  formatChangeTime,
  hasBody,
  sortChanges,
  visibleChanges,
} from "./changes";
import type { ChangeSetItem } from "../types";

function change(overrides: Partial<ChangeSetItem> = {}): ChangeSetItem {
  return {
    commitId: "abc12345",
    message: "feat: something",
    author: "源哥",
    timestamp: 0,
    fileCount: 0,
    body: "",
    ...overrides,
  };
}

const T1 = new Date(2026, 9, 20, 9, 0, 0).getTime();
const T2 = new Date(2026, 9, 21, 11, 30, 0).getTime();
const T3 = new Date(2026, 9, 22, 17, 3, 0).getTime();

describe("sortChanges", () => {
  it("按提交时间倒序：最新的排第一", () => {
    const items = [change({ commitId: "a", timestamp: T1 }), change({ commitId: "c", timestamp: T3 }), change({ commitId: "b", timestamp: T2 })];
    expect(sortChanges(items).map((item) => item.commitId)).toEqual(["c", "b", "a"]);
  });

  it("时间全缺失时保持 Jenkins 原序（旧→新）", () => {
    const items = [change({ commitId: "a", timestamp: 0 }), change({ commitId: "b", timestamp: 0 })];
    expect(sortChanges(items).map((item) => item.commitId)).toEqual(["a", "b"]);
  });

  it("部分缺失时，有时间的排前面，其余保持相对顺序", () => {
    const items = [
      change({ commitId: "no-time-1", timestamp: 0 }),
      change({ commitId: "old", timestamp: T1 }),
      change({ commitId: "no-time-2", timestamp: undefined }),
      change({ commitId: "new", timestamp: T3 }),
    ];
    expect(sortChanges(items).map((item) => item.commitId)).toEqual(["new", "old", "no-time-1", "no-time-2"]);
  });

  it("同一秒的提交保持原来的先后（稳定排序）", () => {
    const items = [change({ commitId: "a", timestamp: T2 }), change({ commitId: "b", timestamp: T2 }), change({ commitId: "c", timestamp: T2 })];
    expect(sortChanges(items).map((item) => item.commitId)).toEqual(["a", "b", "c"]);
  });

  it("不修改传入的数组", () => {
    const items = [change({ commitId: "a", timestamp: T1 }), change({ commitId: "b", timestamp: T2 })];
    sortChanges(items);
    expect(items.map((item) => item.commitId)).toEqual(["a", "b"]);
  });
});

describe("visibleChanges", () => {
  const items = Array.from({ length: 12 }, (_, index) => change({ commitId: `c${index}` }));

  it("未展开时只给前 8 条", () => {
    expect(visibleChanges(items, false, 8)).toHaveLength(8);
  });

  it("展开后给全部", () => {
    expect(visibleChanges(items, true, 8)).toHaveLength(12);
  });

  it("条数不超过上限时原样返回", () => {
    expect(visibleChanges(items.slice(0, 3), false, 8)).toHaveLength(3);
  });

  it("limit 非正数时视为不限", () => {
    expect(visibleChanges(items, false, 0)).toHaveLength(12);
  });
});

describe("changeSummary", () => {
  it("统计提交数、作者数与文件总数", () => {
    const items = [
      change({ author: "源哥", fileCount: 3 }),
      change({ author: "Bob", fileCount: 2 }),
      change({ author: "源哥", fileCount: 1 }),
    ];
    expect(changeSummary(items)).toEqual({ commits: 3, authors: 2, files: 6 });
  });

  it("忽略空作者，缺 fileCount 按 0 算", () => {
    const items = [change({ author: "", fileCount: undefined }), change({ author: "Bob" })];
    expect(changeSummary(items)).toEqual({ commits: 2, authors: 1, files: 0 });
  });

  it("空列表返回全零", () => {
    expect(changeSummary([])).toEqual({ commits: 0, authors: 0, files: 0 });
  });
});

describe("changeBody / hasBody", () => {
  it("正文去掉首尾空白", () => {
    expect(changeBody(change({ body: "\n\n- 明细一\n- 明细二\n" }))).toBe("- 明细一\n- 明细二");
  });

  it("空白正文视为没有正文", () => {
    expect(hasBody(change({ body: "   \n " }))).toBe(false);
    expect(hasBody(change({ body: undefined }))).toBe(false);
  });

  it("有正文时 hasBody 为真", () => {
    expect(hasBody(change({ body: "- 明细" }))).toBe(true);
  });
});

describe("formatChangeTime", () => {
  const now = new Date(2026, 9, 22, 18, 0, 0).getTime();

  it("同年显示月日与时分", () => {
    expect(formatChangeTime(T3, "zh", now)).toBe("10-22 17:03");
  });

  it("跨年显示完整日期", () => {
    const lastYear = new Date(2025, 11, 31, 23, 59, 0).getTime();
    expect(formatChangeTime(lastYear, "zh", now)).toBe("2025-12-31");
  });

  it("英文环境用英文月份缩写", () => {
    expect(formatChangeTime(T3, "en", now)).toBe("Oct 22, 17:03");
  });

  it("缺失时间显示破折号", () => {
    expect(formatChangeTime(0, "zh", now)).toBe("—");
    expect(formatChangeTime(Number.NaN, "zh", now)).toBe("—");
  });
});
