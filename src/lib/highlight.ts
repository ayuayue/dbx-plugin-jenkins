export type CodeLang = "groovy" | "xml";

const HTML_ESCAPES: Record<string, string> = {
  "&": "&amp;",
  "<": "&lt;",
  ">": "&gt;",
};

/** 转义 HTML 特殊字符，保证源码不会被当成标签注入到 v-html。 */
function escapeHtml(text: string): string {
  return text.replace(/[&<>]/g, (ch) => HTML_ESCAPES[ch] ?? ch);
}

function span(cls: string, text: string): string {
  return `<span class="${cls}">${escapeHtml(text)}</span>`;
}

const GROOVY_KEYWORDS = new Set([
  "abstract",
  "as",
  "assert",
  "boolean",
  "break",
  "byte",
  "case",
  "catch",
  "char",
  "class",
  "continue",
  "default",
  "def",
  "do",
  "double",
  "else",
  "enum",
  "extends",
  "false",
  "final",
  "finally",
  "float",
  "for",
  "goto",
  "if",
  "implements",
  "import",
  "in",
  "instanceof",
  "int",
  "interface",
  "long",
  "native",
  "new",
  "null",
  "package",
  "private",
  "protected",
  "public",
  "return",
  "short",
  "static",
  "strictfp",
  "super",
  "switch",
  "synchronized",
  "this",
  "throw",
  "throws",
  "trait",
  "transient",
  "true",
  "try",
  "var",
  "void",
  "volatile",
  "while",
]);

/** Jenkins Pipeline DSL 的常用步骤 / 指令名，单独用 builtin 色区分于语言关键字。 */
const PIPELINE_STEPS = new Set([
  "afterAgent",
  "agent",
  "always",
  "ansiColor",
  "archiveArtifacts",
  "bat",
  "beforeAgent",
  "booleanParam",
  "build",
  "buildDiscarder",
  "catchError",
  "changed",
  "checkout",
  "choice",
  "cleanWs",
  "cleanup",
  "container",
  "credentials",
  "cron",
  "currentBuild",
  "deleteDir",
  "dir",
  "disableConcurrentBuilds",
  "docker",
  "emailext",
  "echo",
  "environment",
  "error",
  "failure",
  "fileExists",
  "fixed",
  "httpRequest",
  "input",
  "junit",
  "load",
  "lock",
  "logRotator",
  "matrix",
  "milestone",
  "node",
  "options",
  "parallel",
  "parameters",
  "password",
  "pipeline",
  "podTemplate",
  "pollSCM",
  "post",
  "powershell",
  "publishHTML",
  "readFile",
  "readJSON",
  "recordIssues",
  "regression",
  "retry",
  "script",
  "sh",
  "skipDefaultCheckout",
  "sleep",
  "slackSend",
  "sshCommand",
  "sshPublisher",
  "stage",
  "stash",
  "steps",
  "string",
  "success",
  "text",
  "timeout",
  "timestamps",
  "tools",
  "triggers",
  "unstable",
  "unstash",
  "upstream",
  "waitUntil",
  "when",
  "withCredentials",
  "withEnv",
  "withGradle",
  "writeFile",
  "writeJSON",
]);

/** 双引号字符串里的 ${...} 单独上色，其余部分仍算字符串。 */
function gstring(raw: string): string {
  let out = "";
  let last = 0;
  const re = /\$\{[^}]*\}/g;
  let match: RegExpExecArray | null = null;
  while ((match = re.exec(raw)) !== null) {
    if (match.index > last) out += span("tok-string", raw.slice(last, match.index));
    out += span("tok-var", match[0]);
    last = match.index + match[0].length;
  }
  if (last < raw.length) out += span("tok-string", raw.slice(last));
  return out;
}

function highlightGroovy(source: string): string {
  const n = source.length;
  let out = "";
  let i = 0;
  let prev = "";

  while (i < n) {
    const ch = source[i]!;
    const next = source[i + 1] ?? "";

    if (ch === "/" && next === "/") {
      const stop = source.indexOf("\n", i);
      const end = stop === -1 ? n : stop;
      out += span("tok-comment", source.slice(i, end));
      i = end;
      continue;
    }

    if (ch === "/" && next === "*") {
      const close = source.indexOf("*/", i + 2);
      const end = close === -1 ? n : close + 2;
      out += span("tok-comment", source.slice(i, end));
      i = end;
      continue;
    }

    if (ch === '"' || ch === "'") {
      const triple = ch.repeat(3);
      const quote = source.startsWith(triple, i) ? triple : ch;
      let j = i + quote.length;
      while (j < n) {
        if (source[j] === "\\") {
          j += 2;
          continue;
        }
        if (source.startsWith(quote, j)) {
          j += quote.length;
          break;
        }
        // 单/双引号字符串不能跨行，遇到换行就把行尾当作结束，剩下的留给下一轮。
        if (quote.length === 1 && source[j] === "\n") break;
        j += 1;
      }
      const end = Math.min(j, n);
      const raw = source.slice(i, end);
      out += ch === '"' ? gstring(raw) : span("tok-string", raw);
      i = end;
      prev = ch;
      continue;
    }

    if (ch >= "0" && ch <= "9") {
      const match = /^(?:0[xX][0-9a-fA-F]+|\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)[lLdDfFgGiI]?/.exec(
        source.slice(i),
      );
      if (match) {
        out += span("tok-number", match[0]);
        i += match[0].length;
        prev = match[0].slice(-1);
        continue;
      }
    }

    if (ch === "$") {
      const match = /^\$\{?[A-Za-z_][\w.]*\}?/.exec(source.slice(i));
      if (match) {
        out += span("tok-var", match[0]);
        i += match[0].length;
        prev = "a";
        continue;
      }
    }

    if (/[A-Za-z_]/.test(ch)) {
      let j = i + 1;
      while (j < n && /[\w$]/.test(source[j]!)) j += 1;
      const word = source.slice(i, j);
      const member = prev === "." || prev === "@";
      if (!member && GROOVY_KEYWORDS.has(word)) out += span("tok-keyword", word);
      else if (!member && PIPELINE_STEPS.has(word)) out += span("tok-builtin", word);
      else out += escapeHtml(word);
      prev = word.slice(-1);
      i = j;
      continue;
    }

    out += escapeHtml(ch);
    if (!/\s/.test(ch)) prev = ch;
    i += 1;
  }

  return out;
}

/** 找到标签结束的 >，跳过属性值引号里的 >。 */
function findTagEnd(source: string, from: number): number {
  let quote = "";
  for (let i = from; i < source.length; i += 1) {
    const ch = source[i]!;
    if (quote) {
      if (ch === quote) quote = "";
      continue;
    }
    if (ch === '"' || ch === "'") {
      quote = ch;
      continue;
    }
    if (ch === ">") return i;
  }
  return -1;
}

function highlightTag(raw: string): string {
  const head = /^<[!/]?[A-Za-z_][\w.:-]*/.exec(raw);
  if (!head) return escapeHtml(raw);

  const tail = raw.endsWith("/>") ? "/>" : raw.endsWith(">") ? ">" : "";
  const inner = tail ? raw.slice(0, raw.length - tail.length) : raw;
  const body = inner.slice(head[0].length);

  // 没有属性的标签整体上色，`</project>` 这一类更好看。
  if (!/\S/.test(body)) return span("tok-tag", raw);

  let out = span("tok-tag", head[0]);
  let i = head[0].length;
  while (i < inner.length) {
    const rest = inner.slice(i);
    const space = /^\s+/.exec(rest);
    if (space) {
      out += space[0];
      i += space[0].length;
      continue;
    }
    const attr = /^[A-Za-z_][\w.:-]*/.exec(rest);
    if (attr) {
      out += span("tok-attr", attr[0]);
      i += attr[0].length;
      const eq = /^\s*=\s*/.exec(inner.slice(i));
      if (eq) {
        out += eq[0];
        i += eq[0].length;
        const value = /^(?:"[^"]*"|'[^']*')/.exec(inner.slice(i));
        if (value) {
          out += span("tok-string", value[0]);
          i += value[0].length;
        }
      }
      continue;
    }
    out += escapeHtml(inner[i]!);
    i += 1;
  }
  if (tail) out += span("tok-tag", tail);
  return out;
}

function highlightXml(source: string): string {
  const n = source.length;
  let out = "";
  let i = 0;

  while (i < n) {
    const lt = source.indexOf("<", i);
    if (lt === -1) {
      out += escapeHtml(source.slice(i));
      break;
    }
    if (lt > i) out += escapeHtml(source.slice(i, lt));

    if (source.startsWith("<!--", lt)) {
      const close = source.indexOf("-->", lt + 4);
      const end = close === -1 ? n : close + 3;
      out += span("tok-comment", source.slice(lt, end));
      i = end;
      continue;
    }

    if (source.startsWith("<?", lt) || source.startsWith("<!", lt)) {
      const closer = source.startsWith("<?", lt) ? "?>" : ">";
      const close = source.indexOf(closer, lt + 2);
      const end = close === -1 ? n : close + closer.length;
      out += span("tok-meta", source.slice(lt, end));
      i = end;
      continue;
    }

    const tag = /^<\/?[A-Za-z_][\w.:-]*/.exec(source.slice(lt));
    if (!tag) {
      out += escapeHtml("<");
      i = lt + 1;
      continue;
    }

    const end = findTagEnd(source, lt);
    if (end === -1) {
      out += escapeHtml(source.slice(lt));
      break;
    }
    out += highlightTag(source.slice(lt, end + 1));
    i = end + 1;
  }

  return out;
}

/**
 * 把源码转成用于 v-html 的高亮片段。
 * 所有源码文本都会先做 HTML 转义，输出的标签只有本函数生成的 span。
 */
export function highlight(source: string, lang: CodeLang): string {
  if (!source) return "";
  return lang === "xml" ? highlightXml(source) : highlightGroovy(source);
}
