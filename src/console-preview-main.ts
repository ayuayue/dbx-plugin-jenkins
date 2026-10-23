import { createApp, h } from "vue";
import ConsoleView from "./components/ConsoleView.vue";
import { locale } from "./i18n";
import "./style.css";

/**
 * 构建详情页「控制台日志列表」预览：`pnpm dev` 打开 /console-preview.html。
 * `?theme=dark` 切深色；`?big=1` 生成 6200 行验证截断与起始行号。
 */
locale.value = "zh";
const search = new URLSearchParams(location.search);
if (search.get("theme") === "dark") document.documentElement.dataset.theme = "dark";
const big = search.get("big") === "1";
/** `?layout=build` 模拟 BuildView 的 flex column + 页面滚动，用来复现底部被裁。 */
const buildLayout = search.get("layout") === "build";

const BODY: string[] = [
  "Started by user demo-user",
  "[Pipeline] Start of Pipeline",
  "[Pipeline] node",
  "Running on Jenkins in /var/jenkins_home/workspace/demo-materials-web",
  "[Pipeline] stage",
  "[Pipeline] { (Build)",
  "[INFO] Scanning for projects...",
  "[INFO] Building demo-materials-web 0.6.0",
  "[INFO] --- maven-compiler-plugin:3.13.0:compile (default-compile) @ demo-materials-web ---",
  "[WARNING] Using platform encoding (UTF-8 actually) to copy filtered resources",
  "[INFO] Compiling 42 source files to /app/target/classes",
  "[INFO] BUILD SUCCESS",
  "[Pipeline] { (Test)",
  "\u001b[1;32m✔ 128 tests passed\u001b[0m",
  "[INFO] --- maven-surefire-plugin:3.2.5:test (default-test) @ demo-materials-web ---",
  "[ERROR] Tests run: 12, Failures: 1, Errors: 0, Skipped: 0",
  "[ERROR] Failed to execute goal org.apache.maven.plugins:maven-surefire-plugin:3.2.5:test",
  "  (default-test) on project demo-materials-web: There are test failures.",
  "Caused by: java.lang.AssertionError: expected:<200> but was:<500>",
  "  at com.example.materials.web.ExportControllerTest.shouldExportOrders(ExportControllerTest.java:88)",
  "  at java.base/jdk.internal.reflect.DirectMethodHandleAccessor.invoke(DirectMethodHandleAccessor.java:103)",
  "  at org.junit.platform.engine.support.hierarchical.NodeTestTask.execute(NodeTestTask.java:95)",
  "[Pipeline] }",
  "[Pipeline] // stage",
  "[Pipeline] stage",
  "[Pipeline] { (Package)",
  "> demo-materials-web@0.6.0 build /app",
  "> vue-tsc --noEmit && vite build",
  "\u001b[33mwarning: unused variable 'legacyFilter' in src/views/order/index.vue\u001b[0m",
  "vite v5.4.21 building for production...",
  "✓ 1842 modules transformed.",
  '[INFO] remote cache {"status":"MISS","key":"maven-compile-8f2c1a4b6d","durationMs":42,"endpoint":"https://repo.example.com/repository/maven-public/org/apache/maven/plugins/maven-compiler-plugin/3.13.0/maven-compiler-plugin-3.13.0.jar","trace":"cache miss, falling back to origin repository with retry budget 3/3"}',
  "dist/index.html                     1.24 kB │ gzip:   0.68 kB",
  "dist/assets/index-8f2c1a.js      1,204.55 kB │ gzip: 356.21 kB",
  "\u001b[1;31m✖ 3 problems (2 errors, 1 warning)\u001b[0m",
  "npm ERR! code ELIFECYCLE",
  "npm ERR! errno 2",
  "npm ERR! demo-materials-web@0.6.0 build: `vue-tsc --noEmit && vite build`",
  "[Pipeline] }",
  "[Pipeline] // stage",
  "[Pipeline] End of Pipeline",
  "Finished: FAILURE",
];

/** `?long=1` 混入大量折行长行，用来验证 content-visibility 的行高估算是否影响「滚到底」。 */
const long = search.get("long") === "1";

/** 生成一份带 ANSI、长行、重复段落的日志，覆盖列表渲染的各个分支。 */
function buildLog(): string {
  const parts: string[] = [];
  const repeats = big || long ? 140 : 1;
  for (let round = 0; round < repeats; round += 1) {
    for (const line of BODY) {
      parts.push(line);
    }
    parts.push(
      `[INFO] flaky-retry round ${round + 1}: waiting for remote cache repo.example.com:443 (attempt ${round + 1}/90)`,
    );
    if (long) {
      for (let index = 0; index < 10; index += 1) {
        parts.push(
          `[INFO] long-${index} r${round}: ${"0123456789abcdef".repeat(24)} [Pipeline] sh`,
        );
      }
    }
  }
  return `${parts.join("\n")}\n`;
}

const FULL = buildLog();

window.dbxPlugin = {
  ready: Promise.resolve(),
  context: {},
  locale: "zh",
  theme: { appearance: "light" },
  capabilities: {},
  async invoke<T>(method: string, params?: Record<string, unknown>): Promise<T> {
    if (method === "jenkins/console") {
      const start = Number(params?.start ?? 0);
      // 预览不模拟增量节奏：一次给完整缓冲区，避免截图时还在续拉。
      const chunk = FULL.slice(start, start + 2_000_000);
      const nextStart = start + chunk.length;
      return {
        text: chunk,
        nextStart,
        moreData: nextStart < FULL.length,
      } as unknown as T;
    }
    throw new Error(`unsupported method in preview: ${method}`);
  },
  async copy(text: string): Promise<void> {
    await navigator.clipboard.writeText(text);
  },
};

const panel = (height: number, label: string) =>
  h(
    "div",
    {
      style: `flex: none; height: ${height}px; background: var(--panel); border: 1px solid var(--border); border-radius: 12px; padding: 16px; color: var(--muted); font-size: 12px;`,
    },
    label,
  );

createApp({
  render: () =>
    h("div", { style: "padding: 20px; max-width: 1180px; margin: 0 auto;" }, [
      h("p", { style: "margin: 0 0 12px; color: var(--muted); font-size: 12px;" }, [
        `控制台预览 · ${big ? "6200 行（截断 + 从 1001 起编号）" : "约 40 行"} · `,
        h("a", { href: "?big=1", style: "color: var(--accent);" }, "大日志"),
        " / ",
        h("a", { href: "?theme=dark", style: "color: var(--accent);" }, "深色"),
        " / ",
        h("a", { href: "?layout=build&big=1", style: "color: var(--accent);" }, "构建页布局"),
      ]),
      buildLayout
        ? h(
            "div",
            {
              style:
                "height: calc(100vh - 80px); display: flex; flex-direction: column; gap: 14px; overflow: auto;",
            },
            [
              panel(150, "构建头部 + 元信息（占位）"),
              panel(260, "变更记录面板（占位）"),
              h(ConsoleView, { job: "demo/demo-materials-web", number: 128 }),
            ],
          )
        : h(ConsoleView, { job: "demo/demo-materials-web", number: 128 }),
    ]),
}).mount("#app");
