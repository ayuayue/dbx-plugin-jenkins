import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { BuildSummary, StageBuildInfo, StageInfo, StageMatrix } from "../types";
import {
  AUTO_REFRESH_INTERVAL_MS,
  PENDING_BUILD_TIMEOUT_MS,
  STAGE_AUTO_REFRESH_INTERVAL_MS,
  createAutoRefresher,
  createPendingBuild,
  evaluatePendingBuild,
  hasNewBuild,
  hasRunningBuild,
  hasRunningStage,
} from "./refresh";

function build(number: number, overrides: Partial<BuildSummary> = {}): BuildSummary {
  return {
    number,
    result: "SUCCESS",
    building: false,
    timestamp: 1700000000000,
    duration: 1000,
    displayName: `#${number}`,
    causes: null,
    status: "success",
    ...overrides,
  };
}

function stage(name: string, status: string): StageInfo {
  return {
    id: name,
    name,
    status,
    startTimeMillis: 1700000000000,
    durationMillis: 1000,
    pauseDurationMillis: 0,
  };
}

function stageBuild(
  number: number,
  overrides: Partial<StageBuildInfo> = {},
): StageBuildInfo {
  return {
    number,
    result: "SUCCESS",
    status: "success",
    timestamp: 1700000000000,
    duration: 1000,
    commitCount: 0,
    stages: [stage("build", "SUCCESS")],
    ...overrides,
  };
}

function matrix(builds: StageBuildInfo[]): StageMatrix {
  return {
    job: "demo",
    stageNames: ["build"],
    builds,
    averages: [],
    averageRun: 0,
  };
}

describe("hasRunningBuild", () => {
  it("returns false for empty input", () => {
    expect(hasRunningBuild([])).toBe(false);
    expect(hasRunningBuild(null)).toBe(false);
    expect(hasRunningBuild(undefined)).toBe(false);
  });

  it("returns false when every build is finished", () => {
    expect(hasRunningBuild([build(2), build(1, { status: "failure" })])).toBe(false);
  });

  it("detects the building flag", () => {
    expect(hasRunningBuild([build(2), build(1, { building: true })])).toBe(true);
  });

  it("detects the building status even without the flag", () => {
    expect(hasRunningBuild([build(1, { status: "building" })])).toBe(true);
  });
});

describe("hasRunningStage", () => {
  it("returns false for missing or empty matrices", () => {
    expect(hasRunningStage(null)).toBe(false);
    expect(hasRunningStage(undefined)).toBe(false);
    expect(hasRunningStage(matrix([]))).toBe(false);
  });

  it("returns false when every build and stage finished", () => {
    expect(hasRunningStage(matrix([stageBuild(2), stageBuild(1)]))).toBe(false);
  });

  it("detects a building build row", () => {
    expect(hasRunningStage(matrix([stageBuild(1, { status: "building" })]))).toBe(true);
  });

  it("detects an in-progress stage regardless of case", () => {
    expect(
      hasRunningStage(matrix([stageBuild(1, { stages: [stage("build", "IN_PROGRESS")] })])),
    ).toBe(true);
    expect(
      hasRunningStage(matrix([stageBuild(1, { stages: [stage("build", "in_progress")] })])),
    ).toBe(true);
  });

  it("detects a building stage status", () => {
    expect(
      hasRunningStage(matrix([stageBuild(1, { stages: [stage("build", "BUILDING")] })])),
    ).toBe(true);
  });

  it("ignores builds without stage details", () => {
    expect(hasRunningStage(matrix([stageBuild(1, { stages: null })]))).toBe(false);
  });
});

describe("refresh intervals", () => {
  it("keeps the build list faster than the stage view", () => {
    expect(AUTO_REFRESH_INTERVAL_MS).toBe(3000);
    expect(STAGE_AUTO_REFRESH_INTERVAL_MS).toBe(5000);
  });
});

describe("createAutoRefresher", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("waits for the interval before the first run and keeps scheduling", async () => {
    const task = vi.fn(async () => {});
    const refresher = createAutoRefresher({ intervalMs: 3000, task });

    refresher.start();
    expect(refresher.active).toBe(true);
    expect(task).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(2999);
    expect(task).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(1);
    expect(task).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(3000);
    expect(task).toHaveBeenCalledTimes(2);

    refresher.stop();
  });

  it("does not re-enter while the previous run is still pending", async () => {
    let resolveTask: (() => void) | undefined;
    const task = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveTask = resolve;
        }),
    );
    const refresher = createAutoRefresher({ intervalMs: 1000, task });

    refresher.start();
    await vi.advanceTimersByTimeAsync(1000);
    expect(task).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(5000);
    expect(task).toHaveBeenCalledTimes(1);

    resolveTask?.();
    await vi.advanceTimersByTimeAsync(0);
    await vi.advanceTimersByTimeAsync(1000);
    expect(task).toHaveBeenCalledTimes(2);

    refresher.stop();
  });

  it("stops scheduling after stop()", async () => {
    const task = vi.fn(async () => {});
    const refresher = createAutoRefresher({ intervalMs: 1000, task });

    refresher.start();
    refresher.stop();
    expect(refresher.active).toBe(false);

    await vi.advanceTimersByTimeAsync(10_000);
    expect(task).not.toHaveBeenCalled();
  });

  it("ignores duplicate start() calls", async () => {
    const task = vi.fn(async () => {});
    const refresher = createAutoRefresher({ intervalMs: 1000, task });

    refresher.start();
    refresher.start();
    await vi.advanceTimersByTimeAsync(1000);
    expect(task).toHaveBeenCalledTimes(1);

    refresher.stop();
  });

  it("can be restarted after stop()", async () => {
    const task = vi.fn(async () => {});
    const refresher = createAutoRefresher({ intervalMs: 1000, task });

    refresher.start();
    await vi.advanceTimersByTimeAsync(1000);
    refresher.stop();
    refresher.start();
    await vi.advanceTimersByTimeAsync(1000);
    expect(task).toHaveBeenCalledTimes(2);

    refresher.stop();
  });

  it("stops once shouldContinue() returns false", async () => {
    const task = vi.fn(async () => {});
    const shouldContinue = vi.fn(() => false);
    const refresher = createAutoRefresher({ intervalMs: 1000, task, shouldContinue });

    refresher.start();
    await vi.advanceTimersByTimeAsync(1000);
    expect(task).toHaveBeenCalledTimes(1);
    expect(refresher.active).toBe(false);

    await vi.advanceTimersByTimeAsync(5000);
    expect(task).toHaveBeenCalledTimes(1);
  });

  it("keeps polling when a run throws", async () => {
    const task = vi
      .fn<() => Promise<void>>()
      .mockRejectedValueOnce(new Error("boom"))
      .mockResolvedValue(undefined);
    const refresher = createAutoRefresher({ intervalMs: 1000, task });

    refresher.start();
    await vi.advanceTimersByTimeAsync(1000);
    expect(task).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(1000);
    expect(task).toHaveBeenCalledTimes(2);

    refresher.stop();
  });

  it("restart() resets the countdown without an extra run", async () => {
    const task = vi.fn(async () => {});
    const refresher = createAutoRefresher({ intervalMs: 3000, task });

    refresher.start();
    await vi.advanceTimersByTimeAsync(2000);
    refresher.restart();
    await vi.advanceTimersByTimeAsync(2000);
    expect(task).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(1000);
    expect(task).toHaveBeenCalledTimes(1);

    refresher.stop();
  });

  it("restart() on an inactive refresher does not start it", async () => {
    const task = vi.fn(async () => {});
    const refresher = createAutoRefresher({ intervalMs: 1000, task });

    refresher.restart();
    await vi.advanceTimersByTimeAsync(5000);
    expect(task).not.toHaveBeenCalled();
    expect(refresher.active).toBe(false);
  });
});

describe("hasNewBuild", () => {
  it("returns false without builds", () => {
    expect(hasNewBuild([], 37)).toBe(false);
    expect(hasNewBuild(null, 37)).toBe(false);
    expect(hasNewBuild(undefined, 37)).toBe(false);
  });

  it("returns false while the newest build is still the previous one", () => {
    expect(hasNewBuild([build(37), build(36)], 37)).toBe(false);
  });

  it("detects a build number greater than the previous one", () => {
    expect(hasNewBuild([build(38), build(37)], 37)).toBe(true);
  });

  it("treats the first ever build as new", () => {
    expect(hasNewBuild([build(1)], 0)).toBe(true);
  });

  it("ignores the previous build even while it is still running", () => {
    expect(hasNewBuild([build(37, { building: true, status: "building" })], 37)).toBe(false);
  });
});

describe("createPendingBuild", () => {
  it("binds the job, the previous build number and a deadline", () => {
    const pending = createPendingBuild("folder/demo", 37, 1_700_000_000_000);

    expect(pending.job).toBe("folder/demo");
    expect(pending.previousNumber).toBe(37);
    expect(pending.deadline).toBe(1_700_000_000_000 + PENDING_BUILD_TIMEOUT_MS);
  });

  it("defaults the deadline to now", () => {
    const before = Date.now();
    const pending = createPendingBuild("demo", 0);
    expect(pending.deadline).toBeGreaterThanOrEqual(before + PENDING_BUILD_TIMEOUT_MS);
  });
});

describe("evaluatePendingBuild", () => {
  const pending = createPendingBuild("demo", 37, 1_700_000_000_000);
  const now = 1_700_000_001_000;

  it("reports stale when the selected job changed", () => {
    expect(evaluatePendingBuild(pending, "other", [build(38)], now)).toBe("stale");
  });

  it("reports settled once a new build shows up", () => {
    expect(evaluatePendingBuild(pending, "demo", [build(38), build(37)], now)).toBe("settled");
  });

  it("prefers settled over expired on the boundary", () => {
    expect(
      evaluatePendingBuild(pending, "demo", [build(38)], pending.deadline + 5000),
    ).toBe("settled");
  });

  it("reports expired when the deadline passed without a new build", () => {
    expect(
      evaluatePendingBuild(pending, "demo", [build(37)], pending.deadline + 1),
    ).toBe("expired");
  });

  it("keeps watching before the deadline", () => {
    expect(evaluatePendingBuild(pending, "demo", [build(37)], now)).toBe("watching");
  });
});
