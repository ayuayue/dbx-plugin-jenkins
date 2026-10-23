import { createApp, h, type VNodeChild } from "vue";
import RefreshBadge from "./components/RefreshBadge.vue";
import { locale } from "./i18n";
import "./style.css";

/**
 * 刷新徽章 / 按钮三态预览页：只用于开发时肉眼检查尺寸与配色，
 * 用 `pnpm dev` 打开 /refresh-preview.html；`?theme=dark` 切深色。
 */
const now = Date.now();

locale.value = "zh";
const theme = new URLSearchParams(location.search).get("theme");
if (theme === "dark") document.documentElement.dataset.theme = "dark";

const label = (text: string) =>
  h("span", { style: "width:200px;color:var(--muted);font-size:12px;flex:none" }, text);

const row = (title: string, node: VNodeChild) =>
  h(
    "div",
    { style: "display:flex;align-items:center;gap:16px;min-height:28px" },
    [label(title), node],
  );

const ghost = (text: string, extra: Record<string, unknown> = {}) =>
  h(
    "button",
    {
      style:
        "padding:8px 13px;border:1px solid var(--border);border-radius:8px;background:transparent;color:var(--text);font-weight:500;font-size:13px",
      ...extra,
    },
    text,
  );

const Harness = {
  setup() {
    return () =>
      h(
        "div",
        {
          style:
            "padding:24px;display:flex;flex-direction:column;gap:14px;font-family:var(--font)",
        },
        [
          h("h2", { style: "font-size:14px;font-weight:500;margin:0" }, "自动刷新徽章"),
          row(
            "轮询中（等下一轮）",
            h(RefreshBadge, {
              mode: "builds",
              paused: false,
              inFlight: false,
              lastAt: now,
              intervalMs: 3000,
            }),
          ),
          row(
            "请求中",
            h(RefreshBadge, {
              mode: "builds",
              paused: false,
              inFlight: true,
              lastAt: now,
              intervalMs: 3000,
            }),
          ),
          row(
            "已暂停",
            h(RefreshBadge, {
              mode: "builds",
              paused: true,
              inFlight: false,
              lastAt: now,
              intervalMs: 3000,
            }),
          ),
          row(
            "无轮询（不渲染）",
            h(RefreshBadge, {
              mode: null,
              paused: false,
              inFlight: false,
              lastAt: now,
              intervalMs: 0,
            }),
          ),
          h(
            "h2",
            { style: "font-size:14px;font-weight:500;margin:10px 0 0" },
            "手动刷新按钮",
          ),
          row("默认", ghost("刷新")),
          row("刷新中", ghost("刷新中…", { disabled: true, style: "padding:8px 13px;border:1px solid var(--border);border-radius:8px;background:transparent;color:var(--muted);font-weight:500;font-size:13px;opacity:.55" })),
          row("已更新", ghost("已更新", { style: "padding:8px 13px;border:1px solid var(--success);border-radius:8px;background:transparent;color:var(--success);font-weight:500;font-size:13px" })),
          h(
            "h2",
            { style: "font-size:14px;font-weight:500;margin:10px 0 0" },
            "构建行变化高亮",
          ),
          h(
            "div",
            {
              style:
                "border:1px solid var(--border);border-radius:8px;overflow:hidden;width:460px",
            },
            [
              h(
                "div",
                {
                  style:
                    "display:flex;gap:18px;padding:8px 10px;font-size:12px;border-bottom:1px solid var(--border);background:color-mix(in srgb, var(--success) 9%, transparent);box-shadow:inset 2px 0 0 var(--success)",
                },
                [
                  h("strong", {}, "#42"),
                  h("span", {}, "构建中"),
                  h("span", { style: "color:var(--muted)" }, "12:29:18"),
                ],
              ),
              h(
                "div",
                { style: "display:flex;gap:18px;padding:8px 10px;font-size:12px" },
                [
                  h("strong", {}, "#41"),
                  h("span", {}, "成功"),
                  h("span", { style: "color:var(--muted)" }, "11:58:02"),
                ],
              ),
            ],
          ),
        ],
      );
  },
};

createApp(Harness).mount("#app");
