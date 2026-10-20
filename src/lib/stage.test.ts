import { describe, expect, it } from "vitest";
import { formatStageDuration, stageIntensity, stageTone } from "./stage";

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
