import { createApp, h, onBeforeUnmount, onMounted, ref } from "vue";
import CodeEditor from "./components/CodeEditor.vue";
import "./style.css";

const sample = `pipeline {
  agent any
  options {
    ansiColor('xterm') // 激活 ANSI 渲染，乱码秒变彩色！
  }
  parameters {
    choice(name: 'MODULE_NAME', choices: ['demo-service'], description: '需要编译的模块')
    booleanParam(name: 'SKIP_CODE_CHECKOUT', defaultValue: false, description: '是否跳过拉取代码过程')
  }
  stages {
    stage('Build') {
      steps {
        sh "mvn -B clean package -pl \${MODULE_NAME}"
      }
    }
  }
}`;

/** 模拟 JobEditor：overlay + modal overflow:hidden + window 级 Escape 监听。 */
const Harness = {
  setup() {
    const open = ref(true);
    const script = ref(sample);
    const onKey = (event: KeyboardEvent): void => {
      if (event.key === "Escape") open.value = false;
    };
    onMounted(() => window.addEventListener("keydown", onKey));
    onBeforeUnmount(() => window.removeEventListener("keydown", onKey));
    return () =>
      open.value
        ? h(
            "div",
            {
              class: "overlay",
              style:
                "position:fixed;inset:0;z-index:30;background:rgba(0,0,0,.42);display:grid;place-items:center;padding:24px",
            },
            [
              h(
                "div",
                {
                  class: "modal",
                  style:
                    "width:680px;max-height:88vh;overflow:hidden;background:#fff;border-radius:12px;padding:16px",
                },
                [
                  h(CodeEditor, {
                    modelValue: script.value,
                    rows: 12,
                    "onUpdate:modelValue": (value: string) => {
                      script.value = value;
                    },
                  }),
                ],
              ),
            ],
          )
        : h("p", { class: "closed" }, "弹窗已关闭");
  },
};

createApp(Harness).mount("#app");
