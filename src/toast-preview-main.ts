import { createApp, h, onMounted, ref } from "vue";
import ToastHost from "./components/ToastHost.vue";
import { createToastQueue, type ToastItem } from "./lib/toast";
import "./style.css";

/**
 * Toast 预览页：只用于开发时肉眼检查浮层样式与堆叠效果，
 * 用 `pnpm dev` 打开 /toast-preview.html。
 */
const Harness = {
  setup() {
    const items = ref<ToastItem[]>([]);
    const queue = createToastQueue({ onChange: (next) => (items.value = next) });

    function fire(text: string, tone: "success" | "info" = "success"): void {
      queue.push(text, tone);
    }

    // 预置三条，方便直接截图核对堆叠与换行。
    onMounted(() => {
      fire("任务已删除。");
      fire("请关闭该连接的只读模式后再管理任务和视图。", "info");
      fire(
        "任务 folder/demo-service-pipeline 已创建，但未能加入视图 release-2026-q4，需要时请到「管理视图」手动添加。",
        "info",
      );
    });

    return () =>
      h("div", { style: "height:100vh;display:flex;flex-direction:column" }, [
        h(
          "div",
          {
            style:
              "height:56px;display:flex;align-items:center;gap:10px;padding:0 18px;background:var(--panel);border-bottom:1px solid var(--border);font-weight:600",
          },
          "Jenkins 管理台",
        ),
        h(
          "div",
          {
            style:
              "flex:1;display:flex;flex-direction:column;gap:10px;align-items:flex-start;padding:24px",
          },
          [
            h(
              "button",
              { style: "padding:6px 12px", onClick: () => fire("任务已删除。") },
              "success 提示",
            ),
            h(
              "button",
              {
                style: "padding:6px 12px",
                onClick: () => fire("请关闭该连接的只读模式后再管理任务和视图。", "info"),
              },
              "info 提示",
            ),
            h(
              "button",
              {
                style: "padding:6px 12px",
                onClick: () =>
                  fire(
                    "任务 folder/demo-service-pipeline 已创建，但未能加入视图 release-2026-q4，需要时请到「管理视图」手动添加。",
                    "info",
                  ),
              },
              "长文案提示",
            ),
          ],
        ),
        h(ToastHost, {
          items: items.value,
          onDismiss: (id: number) => queue.dismiss(id),
        }),
      ]);
  },
};

createApp(Harness).mount("#app");
