import { describe, expect, it } from "vitest";
import { formatDuration, formatTimestamp } from "./format";

describe("formatDuration", () => {
  it("formats sub-minute durations", () => {
    expect(formatDuration(0)).toBe("—");
    expect(formatDuration(-5)).toBe("—");
    expect(formatDuration(1_500)).toBe("1s");
    expect(formatDuration(59_000)).toBe("59s");
  });

  it("formats minutes and hours", () => {
    expect(formatDuration(65_000)).toBe("1m 5s");
    expect(formatDuration(3_600_000)).toBe("1h 0m");
    expect(formatDuration(3_725_000)).toBe("1h 2m");
  });
});

describe("formatTimestamp", () => {
  it("returns a dash for empty values", () => {
    expect(formatTimestamp(0, "zh")).toBe("—");
  });

  it("formats a valid epoch in local time", () => {
    const text = formatTimestamp(1_700_000_000_000, "zh");
    expect(text).not.toBe("—");
    expect(text).toMatch(/\d{4}/);
  });
});
