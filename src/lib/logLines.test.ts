import { describe, expect, it } from "vitest";
import {
  countLevels,
  detectLevel,
  filterLines,
  firstVisibleLine,
  matchRows,
  parseAnsi,
  stripAnsi,
} from "./logLines";

describe("stripAnsi", () => {
  it("removes SGR color sequences", () => {
    expect(stripAnsi("\u001b[31mred\u001b[0m plain")).toBe("red plain");
    expect(stripAnsi("\u001b[1;32mok\u001b[22m done")).toBe("ok done");
  });

  it("removes cursor and erase sequences", () => {
    expect(stripAnsi("\u001b[2K\u001b[1Gprogress 10%")).toBe("progress 10%");
    expect(stripAnsi("\u001b[?25lhidden")).toBe("hidden");
  });

  it("removes OSC title sequences terminated by BEL or ST", () => {
    expect(stripAnsi("\u001b]0;my title\u0007text")).toBe("text");
    expect(stripAnsi("\u001b]8;;https://x.dev\u001b\\link")).toBe("link");
  });

  it("keeps plain text and CJK untouched", () => {
    expect(stripAnsi("构建成功 ✓")).toBe("构建成功 ✓");
    expect(stripAnsi("")).toBe("");
  });
});

describe("parseAnsi", () => {
  it("returns one plain span without escape codes", () => {
    expect(parseAnsi("plain text")).toEqual([{ text: "plain text" }]);
  });

  it("splits colored and default segments", () => {
    expect(parseAnsi("\u001b[31mred\u001b[0m normal")).toEqual([
      { text: "red", tone: "red" },
      { text: " normal" },
    ]);
  });

  it("carries bold and merges adjacent spans with the same style", () => {
    expect(parseAnsi("\u001b[1;32mok\u001b[32m!\u001b[22m x")).toEqual([
      { text: "ok!", tone: "green", bold: true },
      { text: " x", tone: "green" },
    ]);
  });

  it("maps bright colors and ignores unknown SGR codes", () => {
    expect(parseAnsi("\u001b[91mbright\u001b[0m")).toEqual([{ text: "bright", tone: "red" }]);
    expect(parseAnsi("\u001b[4munderline\u001b[0m")).toEqual([{ text: "underline" }]);
  });

  it("drops non-SGR control sequences instead of leaking them", () => {
    expect(parseAnsi("\u001b[2K\u001b[1Gprogress")).toEqual([{ text: "progress" }]);
  });

  it("returns no spans for an empty line", () => {
    expect(parseAnsi("")).toEqual([]);
    expect(parseAnsi("\u001b[0m")).toEqual([]);
  });
});

describe("detectLevel", () => {
  it("flags uppercase level tokens", () => {
    expect(detectLevel("[ERROR] Failed to execute goal")).toBe("error");
    expect(detectLevel("ERROR: something broke")).toBe("error");
    expect(detectLevel("FATAL: cannot continue")).toBe("error");
  });

  it("flags build failures, exceptions and tool-specific errors", () => {
    expect(detectLevel("BUILD FAILED")).toBe("error");
    expect(detectLevel("Finished: FAILURE")).toBe("error");
    expect(detectLevel("java.lang.IllegalStateException: boom")).toBe("error");
    expect(detectLevel("Caused by: java.net.ConnectException")).toBe("error");
    expect(detectLevel("npm ERR! code ELIFECYCLE")).toBe("error");
    expect(detectLevel("src/main.ts(3,7): error TS2322: Type 'string' is not assignable")).toBe(
      "error",
    );
    expect(detectLevel("##[error]Process completed with exit code 1")).toBe("error");
    expect(detectLevel("error: cannot open file")).toBe("error");
  });

  it("flags warnings without treating them as errors", () => {
    expect(detectLevel("[WARNING] Using platform encoding")).toBe("warn");
    expect(detectLevel("WARN deprecated config key")).toBe("warn");
    expect(detectLevel("warning: unused variable")).toBe("warn");
  });

  it("flags plain informational prefixes", () => {
    expect(detectLevel("[INFO] Building jar")).toBe("info");
    expect(detectLevel("  [DEBUG] cache hit")).toBe("info");
  });

  it("does not flag counts or identifiers that merely contain error", () => {
    expect(detectLevel("0 errors, 0 warnings")).toBe("plain");
    expect(detectLevel("errorRate=0.01")).toBe("plain");
    expect(detectLevel("Started by user admin")).toBe("plain");
    expect(detectLevel("")).toBe("plain");
  });

  it("prefers error when a line also contains a warning", () => {
    expect(detectLevel("WARN then ERROR")).toBe("error");
  });
});

describe("countLevels", () => {
  it("counts errors and warnings across lines", () => {
    const lines = ["ERROR: a", "[WARNING] b", "npm ERR! c", "ok", "BUILD FAILED"];
    expect(countLevels(lines)).toEqual({ error: 3, warn: 1 });
  });

  it("returns zeros for empty input", () => {
    expect(countLevels([])).toEqual({ error: 0, warn: 0 });
  });
});

describe("filterLines", () => {
  it("keeps every line and numbers them from firstLineNumber", () => {
    const rows = filterLines(["a", "b"], "all", 1208);
    expect(rows.map((row) => [row.number, row.text])).toEqual([
      [1208, "a"],
      [1209, "b"],
    ]);
  });

  it("keeps only problems while preserving original numbers", () => {
    const rows = filterLines(["ok", "ERROR: bad", "WARN: meh", "ok2"], "problems", 10);
    expect(rows.map((row) => row.number)).toEqual([11, 12]);
    expect(rows.map((row) => row.level)).toEqual(["error", "warn"]);
  });

  it("returns nothing when there is no problem", () => {
    expect(filterLines(["ok"], "problems", 1)).toEqual([]);
  });
});

describe("matchRows", () => {
  it("returns indexes of matching lines, ignoring case", () => {
    expect(matchRows(["ok", "an ERROR here", "error again", "nope"], "error")).toEqual([1, 2]);
  });

  it("counts a line once even when it matches multiple times", () => {
    expect(matchRows(["error error error"], "error")).toEqual([0]);
  });

  it("ignores ANSI escapes when matching", () => {
    expect(matchRows(["\u001b[31merror\u001b[0m: bad"], "error")).toEqual([0]);
  });

  it("returns nothing for a blank query", () => {
    expect(matchRows(["a", "b"], "   ")).toEqual([]);
  });
});

describe("firstVisibleLine", () => {
  it("starts at 1 when nothing was dropped", () => {
    expect(firstVisibleLine(100, 5000)).toBe(1);
    expect(firstVisibleLine(0, 5000)).toBe(1);
  });

  it("accounts for the dropped head", () => {
    expect(firstVisibleLine(5280, 5000)).toBe(281);
    expect(firstVisibleLine(5002, 5000)).toBe(3);
  });
});
