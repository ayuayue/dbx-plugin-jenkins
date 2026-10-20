import { describe, expect, it } from "vitest";
import { highlight } from "./highlight";

const groovy = (src: string): string => highlight(src, "groovy");
const xml = (src: string): string => highlight(src, "xml");

describe("highlight groovy", () => {
  it("returns an empty string for empty input", () => {
    expect(groovy("")).toBe("");
  });

  it("escapes html so source cannot inject markup", () => {
    const html = groovy("echo '<script>alert(1)</script>'");
    expect(html).not.toContain("<script");
    expect(html).toContain("&lt;script&gt;");
  });

  it("highlights line and block comments", () => {
    expect(groovy("// hi")).toContain('<span class="tok-comment">// hi</span>');
    expect(groovy("/* a\nb */")).toContain('<span class="tok-comment">/* a\nb */</span>');
  });

  it("does not treat // inside a string as a comment", () => {
    const html = groovy("sh 'http://x'");
    expect(html).toContain('<span class="tok-string">\'http://x\'</span>');
    expect(html).not.toContain("tok-comment");
  });

  it("highlights keywords, numbers and literals", () => {
    const html = groovy("def count = 42\nreturn true");
    expect(html).toContain('<span class="tok-keyword">def</span>');
    expect(html).toContain('<span class="tok-number">42</span>');
    expect(html).toContain('<span class="tok-keyword">return</span>');
    expect(html).toContain('<span class="tok-keyword">true</span>');
  });

  it("highlights pipeline steps", () => {
    expect(groovy("sh 'make'")).toContain('<span class="tok-builtin">sh</span>');
    expect(groovy("pipeline {\n}")).toContain('<span class="tok-builtin">pipeline</span>');
    expect(groovy("  stage('build') {}")).toContain(
      '<span class="tok-builtin">stage</span>',
    );
  });

  it("highlights double quoted strings and gstring interpolation", () => {
    const html = groovy('echo "job ${env.NAME}"');
    expect(html).toContain('<span class="tok-string">"job ');
    expect(html).toContain('<span class="tok-var">${env.NAME}</span>');
    expect(html).toContain("</span>");
  });

  it("highlights triple quoted strings", () => {
    const html = groovy("echo '''multi\nline'''");
    expect(html).toContain("tok-string");
    expect(html).toContain("line'''");
  });

  it("highlights dollar variables", () => {
    expect(groovy("echo $BUILD_NUMBER")).toContain(
      '<span class="tok-var">$BUILD_NUMBER</span>',
    );
  });

  it("highlights call names followed by a parenthesis", () => {
    expect(groovy("checkout(scm)")).toContain('<span class="tok-builtin">checkout</span>');
  });

  it("does not highlight keywords used as members", () => {
    expect(groovy("obj.def")).not.toContain("tok-keyword");
  });

  it("preserves whitespace and newlines exactly", () => {
    const html = groovy("if (true) {\n  echo 'x'\n}\n");
    expect(html.endsWith("\n")).toBe(true);
    expect(html).toContain("\n  ");
  });
});

describe("highlight xml", () => {
  it("highlights tag names, attributes and values", () => {
    const html = xml('<project name="demo">');
    expect(html).toContain('<span class="tok-tag">&lt;project</span>');
    expect(html).toContain('<span class="tok-attr">name</span>');
    expect(html).toContain('<span class="tok-string">"demo"</span>');
  });

  it("highlights closing tags", () => {
    expect(xml("</project>")).toContain('<span class="tok-tag">&lt;/project&gt;</span>');
  });

  it("highlights comments and declarations", () => {
    expect(xml("<!-- note -->")).toContain(
      '<span class="tok-comment">&lt;!-- note --&gt;</span>',
    );
    expect(xml('<?xml version="1.0"?>')).toContain('<span class="tok-meta">&lt;?xml');
  });

  it("escapes text content that is not a tag", () => {
    expect(xml("<a>1 < 2</a>")).toContain("1 &lt; 2");
  });
});
