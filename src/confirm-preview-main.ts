import { createApp, h, ref } from "vue";
import ConfirmAction from "./components/ConfirmAction.vue";
import { syncLocale, t } from "./i18n";
import "./style.css";

/**
 * 确认弹窗预览页：`pnpm dev` 打开 /confirm-preview.html，`?theme=dark` 切深色。
 * 只用于肉眼核对停止 / 取消排队的二次确认样式，不参与打包。
 */
const theme = new URLSearchParams(location.search).get("theme");
if (theme === "dark") document.documentElement.dataset.theme = "dark";
syncLocale("zh-CN");

const JOB = "folder/demo-service-pipeline";

type Kind = "stop" | "cancel";

const Harness = {
  setup() {
    const kind = ref<Kind>("stop");
    const opening = ref(true);
    const busy = ref(false);

    function open(next: Kind): void {
      kind.value = next;
      opening.value = true;
      busy.value = false;
    }

    // 模拟请求耗时，核对 busy 态下按钮禁用与文案切换。
    function confirm(): void {
      busy.value = true;
      window.setTimeout(() => {
        busy.value = false;
        opening.value = false;
      }, 1200);
    }

    return () =>
      h("div", { style: "padding:24px;display:flex;gap:10px;align-items:center" }, [
        h(
          "button",
          { class: "ghost", type: "button", onClick: () => open("stop") },
          "停止构建",
        ),
        h(
          "button",
          { class: "ghost", type: "button", onClick: () => open("cancel") },
          "取消排队项",
        ),
        opening.value
          ? h(ConfirmAction, {
              title:
                kind.value === "stop"
                  ? t("stopConfirm", { number: 42 })
                  : t("cancelConfirm", { id: "9081" }),
              detail: JOB,
              hint: kind.value === "stop" ? t("stopConfirmHint") : t("cancelConfirmHint"),
              confirmLabel: kind.value === "stop" ? t("stopAction") : t("cancelAction"),
              busy: busy.value,
              onConfirm: confirm,
              onCancel: () => (opening.value = false),
            })
          : null,
      ]);
  },
};

createApp(Harness).mount("#app");
