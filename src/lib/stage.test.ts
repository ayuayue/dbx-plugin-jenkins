import { describe, expect, it } from "vitest";
import {
  commitMessageSummary,
  formatStageDuration,
  popoverPosition,
  shortCommitId,
  stageIntensity,
  stageTone,
} from "./stage";

describe("formatStageDuration", () => {
  it("formats milliseconds below one second", () => {
    expect(formatStageDuration(101)).toBe("101ms");
  });

  it("formats seconds", () => {
    expect(formatStageDuration(1000)).toBe("1s");
    expect(formatStageDuration(1500)).toBe("1.5s");
  });

  it("formats minutes and seconds", () => {
    expect(formatStageDuration(233000)).toBe("3m 53s");
  });

  it("returns a dash for empty durations", () => {
    expect(formatStageDuration(0)).toBe("—");
    expect(formatStageDuration(-1)).toBe("—");
  });
});

describe("stageTone", () => {
  it("maps Jenkins stage statuses", () => {
    expect(stageTone("SUCCESS")).toBe("success");
    expect(stageTone("FAILED")).toBe("danger");
    expect(stageTone("IN_PROGRESS")).toBe("info");
    expect(stageTone("ABORTED")).toBe("muted");
    expect(stageTone("PAUSED_PENDING_INPUT")).toBe("muted");
    expect(stageTone(undefined)).toBe("muted");
  });
});

describe("stageIntensity", () => {
  it("keeps the alpha inside the readable range", () => {
    expect(stageIntensity(0, 1000)).toBeCloseTo(0.1);
    expect(stageIntensity(1000, 1000)).toBeCloseTo(0.45);
    expect(stageIntensity(500, 1000)).toBeCloseTo(0.275);
  });

  it("falls back to the minimum for invalid max", () => {
    expect(stageIntensity(500, 0)).toBeCloseTo(0.1);
  });
});

describe("shortCommitId", () => {
  it("keeps the first eight characters", () => {
    expect(shortCommitId("402ed1d7c9a1b2c3")).toBe("402ed1d7");
  });

  it("keeps short ids untouched", () => {
    expect(shortCommitId("abc123")).toBe("abc123");
  });

  it("trims whitespace and falls back to a dash", () => {
    expect(shortCommitId("  abc123  ")).toBe("abc123");
    expect(shortCommitId("")).toBe("—");
    expect(shortCommitId("   ")).toBe("—");
  });
});

describe("commitMessageSummary", () => {
  it("takes the first non-empty line", () => {
    expect(commitMessageSummary("\n  fix(export): 修正路由注解\n\n详情行")).toBe(
      "fix(export): 修正路由注解",
    );
  });

  it("collapses repeated whitespace", () => {
    expect(commitMessageSummary("chore:   调整   模板")).toBe("chore: 调整 模板");
  });

  it("returns an empty string for blank messages", () => {
    expect(commitMessageSummary("")).toBe("");
    expect(commitMessageSummary("  \n\t ")).toBe("");
  });

  it("truncates long messages by code point", () => {
    const summary = commitMessageSummary("示".repeat(150), 120);
    expect(summary).toBe("示".repeat(120) + "…");
    expect(commitMessageSummary("🚀".repeat(130), 120)).toBe("🚀".repeat(120) + "…");
  });
});

describe("popoverPosition", () => {
  const viewport = { width: 1280, height: 800 };

  it("places the popover to the right of the trigger", () => {
    const position = popoverPosition(
      { top: 100, left: 190, width: 120, height: 40 },
      { width: 360, height: 240 },
      viewport,
    );
    expect(position).toEqual({ top: 100, left: 318 });
  });

  it("flips to the left when the right edge is out of view", () => {
    const position = popoverPosition(
      { top: 100, left: 1100, width: 120, height: 40 },
      { width: 360, height: 240 },
      viewport,
    );
    expect(position.left).toBe(732);
  });

  it("moves up when the bottom edge is out of view", () => {
    const position = popoverPosition(
      { top: 700, left: 190, width: 120, height: 40 },
      { width: 360, height: 240 },
      viewport,
    );
    expect(position.top).toBe(552);
  });

  it("clamps oversized popovers inside the viewport", () => {
    const position = popoverPosition(
      { top: 20, left: 20, width: 120, height: 40 },
      { width: 900, height: 900 },
      { width: 400, height: 400 },
    );
    expect(position).toEqual({ top: 8, left: 8 });
  });
});
