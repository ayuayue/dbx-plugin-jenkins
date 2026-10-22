import { describe, expect, it } from "vitest";
import { countMatches, highlightSegments, tailLines, trimLogTail } from "./console";

describe("tailLines", () => {
  it("splits lines and drops a trailing newline", () => {
    expect(tailLines("a\nb\n", 10)).toEqual(["a", "b"]);
    expect(tailLines("", 10)).toEqual([]);
  });

  it("keeps only the last N lines", () => {
    expect(tailLines("1\n2\n3\n4", 2)).toEqual(["3", "4"]);
    expect(tailLines("1\n2", 10)).toEqual(["1", "2"]);
  });
});

describe("trimLogTail", () => {
  it("leaves short logs untouched", () => {
    expect(trimLogTail("a\nb\n", 100)).toBe("a\nb\n");
    expect(trimLogTail("", 10)).toBe("");
  });

  it("treats a non-positive limit as unlimited", () => {
    expect(trimLogTail("a\nb", 0)).toBe("a\nb");
  });

  it("keeps the tail and drops the partial first line", () => {
    expect(trimLogTail("aaa\nbbb\nccc\n", 8)).toBe("ccc\n");
    expect(trimLogTail("aaa\nbbb\n", 7)).toBe("bbb\n");
  });

  it("falls back to the plain tail when there is no line break", () => {
    expect(trimLogTail("abcdefghij", 4)).toBe("ghij");
  });
});

describe("countMatches", () => {
  it("counts case-insensitive occurrences", () => {
    expect(countMatches(["error: one", "ERROR: two", "ok"], "error")).toBe(2);
    expect(countMatches(["a", "b"], "")).toBe(0);
  });
});

describe("highlightSegments", () => {
  it("splits a line into matched and unmatched parts", () => {
    expect(highlightSegments("a ERROR b error", "error")).toEqual([
      { text: "a ", match: false },
      { text: "ERROR", match: true },
      { text: " b ", match: false },
      { text: "error", match: true },
    ]);
  });

  it("returns a single unmatched segment when there is no query", () => {
    expect(highlightSegments("plain", "")).toEqual([{ text: "plain", match: false }]);
  });

  it("returns the whole line when nothing matches", () => {
    expect(highlightSegments("plain", "zzz")).toEqual([{ text: "plain", match: false }]);
  });
});
