import { describe, expect, it } from "vitest";
import { NAV_STACK_LIMIT, createNavStack, type NavEntry } from "./navStack";

function entry(overrides: Partial<NavEntry> = {}): NavEntry {
  return {
    view: "jobs",
    activeView: "",
    folder: "",
    job: "",
    build: null,
    ...overrides,
  };
}

describe("createNavStack", () => {
  it("初始只有一条记录，前后都退不动", () => {
    const stack = createNavStack(entry());
    expect(stack.size()).toBe(1);
    expect(stack.index()).toBe(0);
    expect(stack.canBack()).toBe(false);
    expect(stack.canForward()).toBe(false);
    expect(stack.back()).toBeNull();
    expect(stack.forward()).toBeNull();
  });

  it("push 之后可以后退，且新导航清空前进分支", () => {
    const stack = createNavStack(entry({ folder: "" }));
    expect(stack.push(entry({ folder: "team-a" }))).toEqual({ index: 1, added: true });
    expect(stack.push(entry({ folder: "team-a/ci" }))).toEqual({ index: 2, added: true });
    expect(stack.canBack()).toBe(true);
    expect(stack.canForward()).toBe(false);

    expect(stack.back()).toBe(1);
    expect(stack.canForward()).toBe(true);

    // 后退状态下再导航：原来 index=2 的记录被截断。
    expect(stack.push(entry({ folder: "team-b" }))).toEqual({ index: 2, added: true });
    expect(stack.size()).toBe(3);
    expect(stack.canForward()).toBe(false);
    expect(stack.current().folder).toBe("team-b");
  });

  it("连续 push 相同条目只保留一条，并标记为未新增", () => {
    const stack = createNavStack(entry());
    stack.push(entry({ job: "demo/alpha" }));
    expect(stack.push(entry({ job: "demo/alpha" }))).toEqual({ index: 1, added: false });
    expect(stack.size()).toBe(2);
  });

  it("相同条目比较覆盖五个字段", () => {
    const stack = createNavStack(entry());
    stack.push(entry({ view: "build", job: "demo/alpha", build: 7 }));
    stack.push(entry({ view: "build", job: "demo/alpha", build: 8 }));
    stack.push(entry({ view: "build", job: "demo/beta", build: 8 }));
    stack.push(entry({ activeView: "ci", view: "build", job: "demo/beta", build: 8 }));
    expect(stack.size()).toBe(5);
  });

  it("超过上限时丢掉最旧的一条，游标同步前移", () => {
    const stack = createNavStack(entry(), 3);
    stack.push(entry({ job: "a" }));
    stack.push(entry({ job: "b" }));
    expect(stack.index()).toBe(2);
    expect(stack.size()).toBe(3);

    expect(stack.push(entry({ job: "c" }))).toEqual({ index: 2, added: true });
    expect(stack.size()).toBe(3);
    expect(stack.index()).toBe(2);
    expect(stack.current().job).toBe("c");
    // 最旧的初始条目已被丢弃，最早一条是 a。
    expect(stack.back()).toBe(1);
    expect(stack.current().job).toBe("b");
    expect(stack.back()).toBe(0);
    expect(stack.current().job).toBe("a");
    expect(stack.back()).toBeNull();
  });

  it("back / forward 到底时游标不动", () => {
    const stack = createNavStack(entry({ job: "a" }));
    stack.push(entry({ job: "b" }));
    expect(stack.back()).toBe(0);
    expect(stack.back()).toBeNull();
    expect(stack.index()).toBe(0);
    expect(stack.forward()).toBe(1);
    expect(stack.forward()).toBeNull();
    expect(stack.index()).toBe(1);
  });

  it("restore 越界时夹紧到边界", () => {
    const stack = createNavStack(entry({ job: "a" }));
    stack.push(entry({ job: "b" }));
    expect(stack.restore(9).job).toBe("b");
    expect(stack.index()).toBe(1);
    expect(stack.restore(-3).job).toBe("a");
    expect(stack.index()).toBe(0);
  });

  it("restore 非数字夹回 0", () => {
    const stack = createNavStack(entry({ job: "a" }));
    stack.push(entry({ job: "b" }));
    expect(stack.restore(Number.NaN).job).toBe("a");
    expect(stack.index()).toBe(0);
  });

  it("reset 清空整栈，只留给定的一条", () => {
    const stack = createNavStack(entry({ job: "a" }));
    stack.push(entry({ job: "b" }));
    stack.reset(entry({ view: "queue", job: "" }));
    expect(stack.size()).toBe(1);
    expect(stack.index()).toBe(0);
    expect(stack.canBack()).toBe(false);
    expect(stack.current().view).toBe("queue");
  });

  it("current 始终返回游标所在条目", () => {
    const stack = createNavStack(entry({ job: "a" }));
    stack.push(entry({ job: "b" }));
    expect(stack.current().job).toBe("b");
    stack.back();
    expect(stack.current().job).toBe("a");
  });

  it("上限至少为 1，非法上限按 1 处理", () => {
    const stack = createNavStack(entry({ job: "a" }), 0);
    stack.push(entry({ job: "b" }));
    expect(stack.size()).toBe(1);
    expect(stack.current().job).toBe("b");
  });

  it("默认上限为 50", () => {
    expect(NAV_STACK_LIMIT).toBe(50);
    const stack = createNavStack(entry());
    for (let i = 0; i < 60; i += 1) stack.push(entry({ job: `job-${i}` }));
    expect(stack.size()).toBe(50);
    expect(stack.current().job).toBe("job-59");
  });
});
