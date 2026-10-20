import { describe, expect, it } from "vitest";
import { gutterText, indentSelection } from "./editor";

describe("gutterText", () => {
  it("numbers every line starting at one", () => {
    expect(gutterText("a\nb\nc")).toBe("1\n2\n3");
  });

  it("counts a single empty line", () => {
    expect(gutterText("")).toBe("1");
  });

  it("counts the trailing empty line", () => {
    expect(gutterText("a\n")).toBe("1\n2");
  });
});

describe("indentSelection", () => {
  it("inserts two spaces at the caret", () => {
    expect(indentSelection("ab", 1, 1)).toEqual({
      value: "a  b",
      selectionStart: 3,
      selectionEnd: 3,
    });
  });

  it("indents every selected line", () => {
    expect(indentSelection("a\nb", 0, 3)).toEqual({
      value: "  a\n  b",
      selectionStart: 0,
      selectionEnd: 7,
    });
  });

  it("skips the last line when the selection ends at a line start", () => {
    expect(indentSelection("a\nb", 0, 2).value).toBe("  a\nb");
  });

  it("outdents every selected line", () => {
    expect(indentSelection("  a\n  b", 0, 7, true).value).toBe("a\nb");
  });

  it("outdents the caret line by at most two spaces", () => {
    expect(indentSelection(" a", 1, 1, true)).toEqual({
      value: "a",
      selectionStart: 0,
      selectionEnd: 0,
    });
  });

  it("keeps text untouched when there is no indentation to remove", () => {
    expect(indentSelection("a", 0, 0, true)).toEqual({
      value: "a",
      selectionStart: 0,
      selectionEnd: 0,
    });
  });

  it("only removes indentation from lines that have it", () => {
    expect(indentSelection("  a\nb", 0, 5, true).value).toBe("a\nb");
  });

  it("removes a leading tab on outdent", () => {
    expect(indentSelection("\ta", 1, 1, true).value).toBe("a");
  });
});
